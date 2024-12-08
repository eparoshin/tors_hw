package main

import (
    "net/http"
    "context"
    "fmt"
    "log"
    "encoding/json"
    "sync/atomic"
    "math/rand"
    "strings"
    "io"
)

type ExternalState struct {
    env *TEnv
    ctx context.Context
    db *Db
    nodes NodesConfig
    nodeId uint64
    roundRobin *atomic.Uint64
}

func (state ExternalState) isLeader() (isLeader bool) {
    state.env.WithLock(func (env *TEnv) {
        isLeader = env.leaderState != nil
    })
    return
}

func (state ExternalState) chooseNextFollower() (NodeConfig) {
    idx := (state.roundRobin.Add(1) - 1) % uint64(len(state.nodes))
    for ; idx == state.nodeId; idx = (state.roundRobin.Add(1) - 1) % uint64(len(state.nodes)) {
    }

    return state.nodes[idx]
}

func getKey(path string) (string, bool) {
    basePath := "/entry/"

    if !strings.HasPrefix(path, basePath) {
        return "", false
    }

    key := strings.TrimPrefix(path, basePath)

    if key == "" {
        return "", false
    }

    return key, true
}

func (state ExternalState) getLeaderOrRandom() (NodeConfig) {
    var leaderId *uint64
    state.env.WithLock(func(env *TEnv) {
        leaderId = env.leaderId
    })

    //if there is no known leader, redirect to random node, maybe it knows the leader
    if leaderId == nil {
        idx := rand.Intn(len(state.nodes))

        for ; idx == int(state.nodeId); idx = rand.Intn(len(state.nodes)) {
        }
        id := uint64(idx)
        leaderId = &id
    }

    return state.nodes[*leaderId]
}

func (state ExternalState) redirectToFollower(w http.ResponseWriter, r *http.Request) {
    node := state.chooseNextFollower()
    uri := node.ExternalUri() + r.URL.Path
    http.Redirect(w, r, uri, http.StatusSeeOther)
}

func (state ExternalState) redirectToLeader(w http.ResponseWriter, r *http.Request) {
    node := state.getLeaderOrRandom()
    uri := node.ExternalUri() + r.URL.Path
    http.Redirect(w, r, uri, http.StatusTemporaryRedirect)
}

type KeyVal struct {
    Key string `json:"key"`
    Value string `json:"value"`
}

func (state ExternalState) handleGet(w http.ResponseWriter, r *http.Request) {
    if state.isLeader() {
        state.redirectToFollower(w, r)
        return
    }

    if key, ok := getKey(r.URL.Path); ok {
        if val, found := state.db.Get(key); found {
            getResp := KeyVal{key, val}
            resp, err := json.Marshal(getResp)
            if err != nil {
                log.Fatal(err)
            }
            n, err := w.Write(resp)
            if err != nil || n < len(resp) {
                log.Printf("Error while writing response: %v, %d bytes written", err, n)
                return
            }
        } else {
            resp, err := json.Marshal(map[string]string{"error": "Key not found"})
            if err != nil {
                log.Fatal(err)
            }

            w.WriteHeader(http.StatusNotFound)
            n, err := w.Write(resp)
            if err != nil || n < len(resp) {
                log.Printf("Error while writing response: %v, %d bytes written", err, n)
                return
            }

        }
    } else {
        http.Error(w, "Not found", http.StatusNotFound)
    }

}

func (state ExternalState) handleCreate(w http.ResponseWriter, r *http.Request) {
    if !state.isLeader() {
        state.redirectToLeader(w, r)
        return
    }

    var createRequest KeyVal
    data, err := io.ReadAll(r.Body)
    if err != nil {
        log.Printf("Error while reading req body: %v", err)
        return
    }

    if err = json.Unmarshal(data, &createRequest); err != nil {
        http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
        return
    }

    if state.env.ApplyRequestSync(CREATE, createRequest.Key, createRequest.Value, "") {
        resp, err := json.Marshal(map[string]string{"message": "Entry created successfully"})
        if err != nil {
            log.Fatal(err)
        }

        w.WriteHeader(http.StatusCreated)
        n, err := w.Write(resp)
        if err != nil || n < len(resp) {
            log.Printf("Error while writing response: %v, %d bytes written", err, n)
            return
        }

    } else {
        resp, err := json.Marshal(map[string]string{"error": "Entry already exists"})
        if err != nil {
            log.Fatal(err)
        }

        w.WriteHeader(http.StatusConflict)
        n, err := w.Write(resp)
        if err != nil || n < len(resp) {
            log.Printf("Error while writing response: %v, %d bytes written", err, n)
            return
        }

    }

}

func (state ExternalState) handleUpdateOrCas(w http.ResponseWriter, r *http.Request) {
    if !state.isLeader() {
        state.redirectToLeader(w, r)
        return
    }

    var updateRequest struct {
        PrevValue *string `json:"prev_value"`
        Value string `json:"value"`
    }

    data, err := io.ReadAll(r.Body)
    if err != nil {
        log.Printf("Error while reading req body: %v", err)
        return
    }

    if err = json.Unmarshal(data, &updateRequest); err != nil {
        http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
        return
    }

    applyRequestSync := func (key string) bool {
        if updateRequest.PrevValue != nil {
            return state.env.ApplyRequestSync(CAS, key, updateRequest.Value, *updateRequest.PrevValue)
        } else {
            return state.env.ApplyRequestSync(UPDATE, key, updateRequest.Value, "")
        }
    }

    if key, ok := getKey(r.URL.Path); ok && applyRequestSync(key) {
        resp, err := json.Marshal(map[string]string{"message": "Entry updated successfully"})
        if err != nil {
            log.Fatal(err)
        }

        n, err := w.Write(resp)
        if err != nil || n < len(resp) {
            log.Printf("Error while writing response: %v, %d bytes written", err, n)
            return
        }

    } else {
        resp, err := json.Marshal(map[string]string{"error": "Entry not found"})
        if err != nil {
            log.Fatal(err)
        }

        w.WriteHeader(http.StatusNotFound)
        n, err := w.Write(resp)
        if err != nil || n < len(resp) {
            log.Printf("Error while writing response: %v, %d bytes written", err, n)
            return
        }

    }
}

func (state ExternalState) handleDelete(w http.ResponseWriter, r *http.Request) {
    if !state.isLeader() {
        state.redirectToLeader(w, r)
        return
    }

    if key, ok := getKey(r.URL.Path); ok && state.env.ApplyRequestSync(DELETE, key, "", "") {
        resp, err := json.Marshal(map[string]string{"message": "Entry deleted successfully"})
        if err != nil {
            log.Fatal(err)
        }

        n, err := w.Write(resp)
        if err != nil || n < len(resp) {
            log.Printf("Error while writing response: %v, %d bytes written", err, n)
            return
        }

    } else {
        resp, err := json.Marshal(map[string]string{"error": "Entry not found"})
        if err != nil {
            log.Fatal(err)
        }

        w.WriteHeader(http.StatusNotFound)
        n, err := w.Write(resp)
        if err != nil || n < len(resp) {
            log.Printf("Error while writing response: %v, %d bytes written", err, n)
            return
        }

    }
}

func returnNotAllowed(w http.ResponseWriter) {
    w.WriteHeader(http.StatusMethodNotAllowed)
    w.Header().Add("Allow", "GET, PUT, DELETE")
}

func (state ExternalState) handleEntry(w http.ResponseWriter, r *http.Request) {
    switch (r.Method) {
    case "GET":
        state.handleGet(w, r)
    case "PUT":
        state.handleUpdateOrCas(w, r)
    case "DELETE":
        state.handleDelete(w, r)
    default:
        returnNotAllowed(w)
    }

}

func NewExtServer(env *TEnv, db *Db, ctx context.Context, nodesConfig NodesConfig, nodeId uint64, appConfig AppConfig) (*http.Server, error) {

    state := ExternalState{
        env: env,
        ctx: ctx,
        db: db,
        nodes: nodesConfig,
        nodeId: nodeId,
        roundRobin: &atomic.Uint64{},
    }

    serveMux := http.NewServeMux()
    serveMux.HandleFunc("/entry", state.handleCreate)
    serveMux.HandleFunc("/entry/", state.handleEntry)

    return &http.Server {
        Addr:           fmt.Sprintf(":%d", nodesConfig[nodeId].ExternalPort),
        Handler:        serveMux,
    }, nil
}

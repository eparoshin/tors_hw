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
    basePath = "/entry/"

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
        idx := rand.Uint64n(len(state.nodes))

        for ; idx == state.nodeId; idx = rand.Uint64(len(state.nodes)) {
        }
        leaderId = &idx
    }

    return state.nodes[*leaderId]
}

func (state ExternalState) redirectToFollower(w http.ResponseWriter, r *http.Request) {
    node := state.chooseNextFollower()
    uri := node.ExternalUri() + "/" + r.URL.Path
    http.Redirect(w, r, uri, http.StatusSeeOther)
}

func (state ExternalState) redirectToLeader(w http.ResponseWriter, r *http.Request) {
    node := state.getLeaderOrRandom()
    uri := node.ExternalUri() + "/" + r.URL.Path
    http.Redirect(w, r, uri, http.StatusTemporaryRedirect)
}

type GetResponse struct {
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
            getResp := GetResponse{key, val}
            resp, err := json.Marshal(voteResponse)
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

            n, err := w.Write(resp)
            if err != nil || n < len(resp) {
                log.Printf("Error while writing response: %v, %d bytes written", err, n)
                return
            }

            w.WriteCode(http.StatusNotFound)
        }
    } else {
        http.error(w, "Not found", http.StatusNotFound)
    }

}

func (state ExternalState) handleCreate(w http.ResponseWriter, r *http.Request) {
    if !state.isLeader() {
        state.redirectToLeader(w, r)
        return
    }
}

func (state ExternalState) handleUpdate(w http.ResponseWriter, r *http.Request) {
    if !state.isLeader() {
        state.redirectToLeader(w, r)
        return
    }
}

func (state ExternalState) handleDelete(w http.ResponseWriter, r *http.Request) {
    if !state.isLeader() {
        state.redirectToLeader(w, r)
        return
    }
}

func returnNotAllowed(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Allow", "GET, POST, PUT, DELETE")
    w.WriteHeader(http.StatusMethodNotAllowed)
}

func (state ExternalState) HandleEntry(w http.ResponseWriter, r *http.Request) {
    switch (r.Method) {
    case "GET":
        state.handleGet(w, r)
    case "POST":
        state.handleCreate(w, r)
    case "PUT":
        state.handleUpdate(w, r)
    case "DELETE":
        state.handleDelete(w, r)
    default:
        returnNotAllowed(w, r)


    }
    if state.isLeader() {
        state.redirectToFollower(w, r)
    } else {
        state.handleFollowerGet(w, r)

    }


}

func NewExtServer(env *TEnv, db *Db, ctx context.Context, nodesConfig NodesConfig, nodeId uint64, appConfig AppConfig) (*http.Server, error) {

    state := ExternalState{
        env: env,
        ctx: ctx,
        db: db,
        nodes: nodes,
        nodeId: nodeId
        roundRobin: &atomic.Uint64{},
    }

    //go raftState.periodicCheckHb()

    serveMux := http.NewServeMux()
    serveMux.HandleFunc("/entry/", state.HandleEntry)

    return &http.Server {
        Addr:           fmt.Sprintf(":%d", nodesConfig[nodeId].ExternalPort),
        Handler:        serveMux,
    }, nil
}

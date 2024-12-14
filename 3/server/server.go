package main

import (
    "fmt"
    "log"
    "encoding/json"
    "net/http"
    "strings"
    "context"
    "io"
    "time"
    "bytes"
)

type TState struct  {
    Ctx context.Context
    Config TConfig
    Db *TDb
    Clock *TClock
}

func getKey(path string) (string, bool) {
    basePath := "/get/"

    if !strings.HasPrefix(path, basePath) {
        return "", false
    }

    key := strings.TrimPrefix(path, basePath)

    if key == "" {
        return "", false
    }

    return key, true
}

func broadcastToNode(parentCtx context.Context, node string, key string, entry TVal) {
    timeout := time.Second
    numTries := 0
    tryFunc := func () bool {
        log.Printf("Start try to node %s try num %d\n", node, numTries)
        numTries += 1
        ctx, cancelFunc := context.WithTimeout(parentCtx, timeout)
        defer cancelFunc()
        bcRequest := BroadcastRequest{
            Key: key,
            Val: entry,
        }

        reqBody, err := json.Marshal(bcRequest)
        if err != nil {
            log.Fatal(err)
        }

        request, err := http.NewRequestWithContext(ctx, "POST", node + "/broadcast", bytes.NewReader(reqBody))
        resp, err := http.DefaultClient.Do(request)
        if err != nil {
            log.Print(err)
        }
        resp.Body.Close()

        return err == nil && resp.StatusCode / 100 == 2
    }

    for ; !tryFunc(); {
        time.Sleep(timeout)
    }
}

func (state TState) broadcastToAll(key string, entry TVal) {
    for _, node := range state.Config {
        go broadcastToNode(state.Ctx, node, key, entry)
    }
}

type PatchEntry struct {
    Key string `json:"key"`
    Val string `json:"val"`
}

func (state TState) HandlePatch(w http.ResponseWriter, r *http.Request) {
    var patchRequest []PatchEntry
    data, err := io.ReadAll(r.Body)

    if err != nil {
        log.Printf("Error while reading req body: %v", err)
        return
    }

    if err = json.Unmarshal(data, &patchRequest); err != nil {
        http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
        return
    }

    newTs := func () TTimestamp {
       state.Clock.M.Lock()
       defer state.Clock.M.Unlock()
       state.Clock.Ts.Time += 1
       return state.Clock.Ts
    }()

    for _, entry := range patchRequest {
        tval := TVal{Val: entry.Val, Ts: newTs, Deleted: false,}
        state.broadcastToAll(entry.Key, tval)
    }

}

type DeleteEntry struct {
    Key string `json:"key"`
}

func (state TState) HandleDelete(w http.ResponseWriter, r *http.Request) {
    var deleteRequest []DeleteEntry
    data, err := io.ReadAll(r.Body)

    if err != nil {
        log.Printf("Error while reading req body: %v", err)
        return
    }

    if err = json.Unmarshal(data, &deleteRequest); err != nil {
        http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
        return
    }

    newTs := func () TTimestamp {
       state.Clock.M.Lock()
       defer state.Clock.M.Unlock()
       state.Clock.Ts.Time += 1
       return state.Clock.Ts
    }()

    for _, entry := range deleteRequest {
        tval := TVal{Ts: newTs, Deleted: true,}
        state.broadcastToAll(entry.Key, tval)
    }
}

func returnNotAllowed(w http.ResponseWriter) {
    w.WriteHeader(http.StatusMethodNotAllowed)
    w.Header().Add("Allow", "PATCH, DELETE")
}

func (state TState) HandleChange(w http.ResponseWriter, r *http.Request) {
    switch (r.Method) {
    case "PATCH":
        state.HandlePatch(w, r)
    case "DELETE":
        state.HandleDelete(w, r)
    default:
        returnNotAllowed(w)
    }
}

func (state TState) HandleGet(w http.ResponseWriter, r *http.Request) {
    if key, ok := getKey(r.URL.Path); ok {
        if val, ok := state.Db.Get(key); ok {
            resp, err := json.Marshal(map[string]string{key: val})
            if err != nil {
                log.Fatal(err)
            }
            n, err := w.Write(resp)
            if err != nil || n < len(resp) {
                log.Printf("Error while writing response: %v, %d bytes written", err, n)
            }
        } else {
            http.Error(w, "Not found", http.StatusNotFound)
        }
    } else {
        http.Error(w, "Not found", http.StatusNotFound)
    }
}

type BroadcastRequest struct {
    Key string `json:"key"`
    Val TVal `json:"tval"`
}

func (state TState) HandleBroadcast(w http.ResponseWriter, r *http.Request) {

    var bcRequest BroadcastRequest
    data, err := io.ReadAll(r.Body)

    if err != nil {
        log.Printf("Error while reading req body: %v", err)
        return
    }

    if err = json.Unmarshal(data, &bcRequest); err != nil {
        http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
        return
    }

    if bcRequest.Val.Deleted {
        state.Db.Delete(bcRequest.Key, bcRequest.Val.Ts)
    } else {
        state.Db.Put(bcRequest.Key, bcRequest.Val.Val, bcRequest.Val.Ts)
    }
}

func NewServer(ctx context.Context, config TConfig, port int, db *TDb, clock *TClock) (*http.Server, error) {
    state := TState {
        Ctx: ctx,
        Config: config,
        Db: db,
        Clock: clock,
    }

    serveMux := http.NewServeMux()
    serveMux.HandleFunc("/entries", state.HandleChange)
    serveMux.HandleFunc("/get/", state.HandleGet)
    serveMux.HandleFunc("/broadcast", state.HandleBroadcast)

    return &http.Server {
        Addr:           fmt.Sprintf(":%d", port),
        Handler:        serveMux,
    }, nil


}

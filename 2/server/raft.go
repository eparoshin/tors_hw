package main

import (
    "net/http"
    "context"
    "fmt"
    "encoding/json"
    "io"
    "log"
    "time"
    "sync/atomic"
    "math/rand"
    "bytes"
    "sync"
)

type RaftState struct {
    env *TEnv
    ctx context.Context
    nodesConfig NodesConfig
    nodeId uint64
    appConfig AppConfig
    gotHb *atomic.Bool
    isLeader *atomic.Bool
}

type VoteRequest struct {
    Term uint64 `json:"term"`
    CandidateId uint64 `json:"candidate_id"`
    LastLogIndex uint64 `json:"last_log_index"`
    LastLogTerm uint64 `json:"last_log_term"`
}

type VoteResponse struct {
    Term uint64 `json:"term"`
    VoteGranted bool `json:"vote_granted"`
}

func (state RaftState) HandleRequestVote(w http.ResponseWriter, r *http.Request) {
    var voteRequest VoteRequest
    data, err := io.ReadAll(r.Body)
    if err != nil {
        log.Printf("Error while reading req body: %v", err)
        w.WriteHeader(500)
        return
    }

    if err = json.Unmarshal(data, &voteRequest); err != nil {
        http.Error(w, fmt.Sprint(err), 400)
        return
    }

    var voteResponse VoteResponse
    state.env.WithLock(func(env *TEnv) {
        if voteRequest.Term < env.p.State.CurrentTerm {
            voteResponse.Term = env.p.State.CurrentTerm
            voteResponse.VoteGranted = false
            return
        }
        if voteRequest.Term > env.p.State.CurrentTerm {
            env.p.State.CurrentTerm = voteRequest.Term
            env.leaderState = nil
            env.p.State.VotedFor = nil
            env.p.DumpPState()
        }

        voteResponse.Term = env.p.State.CurrentTerm

        if env.p.State.VotedFor != nil {
            voteResponse.VoteGranted = false
            return
        }

        if voteRequest.LastLogTerm > env.l.Back().Term {
            voteResponse.VoteGranted = true
            env.p.SetVote(voteRequest.CandidateId)
        } else if voteRequest.LastLogTerm == env.l.Back().Term {
            if voteRequest.LastLogIndex >= uint64(len(env.l.Entries)) {
                voteResponse.VoteGranted = true
                env.p.SetVote(voteRequest.CandidateId)
            } else {
                voteResponse.VoteGranted = false
            }
        } else {
            voteResponse.VoteGranted = false
        }
    })

    log.Printf("VoteRequest: %v \n VoteResponse: %v", voteRequest, voteResponse)

    resp, err := json.Marshal(voteResponse)
    if err != nil {
        log.Fatal(err)
    }

    n, err := w.Write(resp)
    if err != nil || n < len(resp) {
        log.Printf("Error while writing response: %v, %d bytes written", err, n)
        return
    }
}

func (state RaftState) requestVoteFrom(ctx context.Context, node NodeConfig, votedChan chan <- VoteResponse) {
    voteRequest := VoteRequest{
        Term: state.env.p.State.CurrentTerm,
        CandidateId: state.nodeId,
        LastLogIndex: uint64(len(state.env.l.Entries)),
        LastLogTerm: state.env.l.Back().Term,
    }


    body, err := json.Marshal(voteRequest)
    if err != nil {
        log.Fatal(err)
    }

    request, err := http.NewRequestWithContext(ctx, "POST", node.InternalUri() + "/request_vote", bytes.NewReader(body))
    if err != nil {
        log.Fatal(err)
    }
    resp, err := http.DefaultClient.Do(request)
    if err != nil {
        log.Print(err)
        return
    }

    var voteResponse VoteResponse
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Print(err)
        return
    }
    resp.Body.Close()

    if resp.StatusCode / 200 != 1 {
        log.Printf("Non Ok response from node: %d, %s\n", resp.StatusCode, resp.Status)
        return
    }

    if err = json.Unmarshal(respBody, &voteResponse); err != nil {
        log.Print(err)
        return
    }

    votedChan <- voteResponse
    return

}

func (state RaftState) leaderHB(ctx context.Context, nodeId uint64, node NodeConfig) {
    log.Println("Start leaderHB to node ", node)
    //TODO
}

func (state RaftState) periodicLeaderHB() {
    hbPeriod := time.Duration(int64(state.appConfig.HBIntervalMs)) * time.Millisecond
    ticker := time.NewTicker(hbPeriod)
    for {
        select {
        case <- ticker.C:
            if (state.isLeader.Load()) {
                func() {
                    requestsTimeout := time.Duration(int64(state.appConfig.AppendEntriesTimeoutMs)) * time.Millisecond
                    ctx, cancelFunc := context.WithTimeout(state.ctx, requestsTimeout)
                    defer cancelFunc()
                    var wg sync.WaitGroup
                    for i, node := range state.nodesConfig {
                        if i == int(state.nodeId) {
                            continue
                        }

                        wg.Add(1)
                        go func (wg *sync.WaitGroup, i uint64, node NodeConfig) {
                            state.leaderHB(ctx, i, node)
                            wg.Done()
                        }(&wg, uint64(i), node)
                    }
                    wg.Done()
                }()
            } else {
                log.Println("I am not leader anymore")
                return
            }
        case <- state.ctx.Done():
            log.Println("Finished periodic leader hb")
            return
        }
    }
}

func (state RaftState) AlreadyLeader() (alreadyLeader bool) {
    state.env.WithLock(func(env *TEnv) {
        alreadyLeader = env.leaderState != nil
    })
    return
}

func (state RaftState) TryBecomeLeader() {
    if state.AlreadyLeader() {
        return
    }

    state.env.WithLock(func(env *TEnv) {
        votedChan := make(chan VoteResponse, len(state.nodesConfig))
        env.p.State.CurrentTerm += 1
        env.p.State.VotedFor = &state.nodeId
        env.p.DumpPState()
        votedChan <- VoteResponse{VoteGranted: true,} //vote for myself

        requestsTimeout := time.Duration(int64(state.appConfig.VoteRequestTimeoutMs)) * time.Millisecond
        ctx, cancelFunc := context.WithTimeout(state.ctx, requestsTimeout)
        for i, node := range state.nodesConfig {
            if i == int(state.nodeId) {
                continue
            }

            go state.requestVoteFrom(ctx, node, votedChan)
        }

        becameLeader := func() bool {
            defer cancelFunc()
            trueCount := 0
            falseCount := 0
            for {
                select {
                case <-ctx.Done():
                    log.Print("Vote Requests timed out")
                    return false
                case resp := <- votedChan:
                    if resp.VoteGranted {
                        trueCount += 1
                    } else {
                        falseCount += 1
                    }

                    if resp.Term > env.p.State.CurrentTerm {
                        env.p.State.CurrentTerm = resp.Term
                        env.p.State.VotedFor = nil
                        env.p.DumpPState()
                        return false
                    }

                    if trueCount > len(state.nodesConfig) / 2 {
                        return true
                    }
                    if falseCount > len(state.nodesConfig) / 2 {
                        return false
                    }
                }

            }
        }()

        if becameLeader {
            log.Printf("I (nodeId: %d) became leader in term %d\n", state.nodeId, env.p.State.CurrentTerm)
            state.isLeader.Store(true)

            env.leaderId = &state.nodeId
            env.leaderState = NewLeaderState(len(state.nodesConfig), uint64(len(env.l.Entries)))

            go state.periodicLeaderHB()
        }
    })

}

type AppendRequest struct {
    Term uint64 `json:"term"`
    LeaderId uint64 `json:"leader_id"`
    PrevLogIndex uint64 `json:"prev_log_index"`
    PrevLogTerm uint64 `json:"prev_log_term"`
    Entries []LogEntry `json:"entries"`
    LeaderCommit uint64 `json:"leader_commit"`
}

type AppendResponse struct {
    Term uint64 `json:"term"`
    Success bool `json:"success"`
}

func (state RaftState) HandleAppendEntries(w http.ResponseWriter, r *http.Request) {
    var appendRequest AppendRequest
    data, err := io.ReadAll(r.Body)
    if err != nil {
        log.Printf("Error while reading req body: %v", err)
        w.WriteHeader(500)
        return
    }

    if err = json.Unmarshal(data, &appendRequest); err != nil {
        http.Error(w, fmt.Sprint(err), 400)
        return
    }

    var appendResponse AppendResponse
    state.env.WithLock(func(env *TEnv) {
        if appendRequest.Term < env.p.State.CurrentTerm {
            appendResponse.Term = env.p.State.CurrentTerm
            appendResponse.Success = false
            return
        } else {
            state.gotHb.Store(true)
            env.leaderState = nil
            env.leaderId = &appendRequest.LeaderId
            env.p.State.CurrentTerm = appendRequest.Term
            env.p.State.VotedFor = &appendRequest.LeaderId
            env.p.DumpPState()
        }


        if (!env.l.CheckAndCorrect(appendRequest.PrevLogIndex, appendRequest.PrevLogTerm)) {
            appendResponse.Term = env.p.State.CurrentTerm
            appendResponse.Success = false
            return
        }

        env.l.AppendEntries(appendRequest.Entries)

        env.CommitChanges(appendRequest.LeaderCommit)

        appendResponse.Term = env.p.State.CurrentTerm
        appendResponse.Success = true

    })

    log.Printf("AppendRequest: %v \n AppendResponse: %v", appendRequest, appendResponse)

    resp, err := json.Marshal(appendResponse)
    if err != nil {
        log.Fatal(err)
    }

    n, err := w.Write(resp)
    if err != nil || n < len(resp) {
        log.Printf("Error while writing response: %v, %d bytes written", err, n)
        return
    }
}

func calcDeadline(durationMs int, randomShiftMs int) time.Duration {
    return time.Duration(int64(durationMs + rand.Intn(randomShiftMs))) * time.Millisecond
}

func (state RaftState) periodicCheckHb() {
    log.Print("Periodic check heartbeat started")
    for {
        timer := time.NewTimer(calcDeadline(state.appConfig.HBTimeout, state.appConfig.RandomShift))
        select {
            case <- timer.C:
                if !state.gotHb.Swap(false) {
                    log.Print("No heartbeats, initiate revote")
                    state.TryBecomeLeader()
                }
            case <- state.ctx.Done():
                log.Print("Periodic check heartbeat exited")
                return
        }
    }
}

func NewRaftServer(env *TEnv, ctx context.Context, nodesConfig NodesConfig, nodeId uint64, appConfig AppConfig) (*http.Server, error) {
    raftState := RaftState{
        env: env,
        ctx: ctx,
        nodesConfig: nodesConfig,
        nodeId: nodeId,
        appConfig: appConfig,
        gotHb: &atomic.Bool{},
        isLeader: &atomic.Bool{},
    }

    go raftState.periodicCheckHb()

    serveMux := http.NewServeMux()
    serveMux.HandleFunc("/request_vote", raftState.HandleRequestVote)
    serveMux.HandleFunc("/append_entries", raftState.HandleAppendEntries)

    return &http.Server {
        Addr:           fmt.Sprintf(":%d", nodesConfig[nodeId].InternalPort),
        Handler:        serveMux,
    }, nil
}

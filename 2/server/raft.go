package main

import (
    "net/http"
    "context"
    "fmt"
    "encoding/json"
    "io"
    "log"
)

type RaftState struct {
    env *TEnv
    ctx context.Context
    nodesConfig NodesConfig
    nodeId uint64
    appConfig AppConfig
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
        }
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

func NewRaftServer(env *TEnv, ctx context.Context, nodesConfig NodesConfig, nodeId uint64, appConfig AppConfig) (*http.Server, error) {
    raftState := RaftState{
        env: env,
        ctx: ctx,
        nodesConfig: nodesConfig,
        nodeId: nodeId,
        appConfig: appConfig,
    }

    serveMux := http.NewServeMux()
    serveMux.HandleFunc("/request_vote", raftState.HandleRequestVote)
    serveMux.HandleFunc("/append_entries", raftState.HandleAppendEntries)

    return &http.Server {
        Addr:           fmt.Sprintf(":%d", nodesConfig[nodeId].InternalPort),
        Handler:        serveMux,
    }, nil
}

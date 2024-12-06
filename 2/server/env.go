package main

import (
    "sync"
    "time"
)

type LeaderState struct {
    NextIndex []uint64
    MatchIndex []uint64
}

func NewLeaderState(nodeId uint64, numNodes int, lastLogIndex uint64) *LeaderState {
    nextIndex := make([]uint64, numNodes)
    for i := range nextIndex {
        nextIndex[i] = lastLogIndex + 1
    }

    state := LeaderState {
        NextIndex: nextIndex,
        MatchIndex: make([]uint64, numNodes),
    }
    state.MatchIndex[nodeId] = ^uint64(0)
    return &state
}

type TEnv struct {
    p PState
    l Log
    commitIndex uint64
    lastApplied uint64
    leaderState *LeaderState
    leaderId *uint64
    lastHB time.Time
    commitQueue chan LogEntry
    newEntriesAlert Alert
    m sync.Mutex
}

func NewEnv(p PState, l Log, logQueueSize uint) TEnv {
    return TEnv{p: p, l: l, commitQueue: make(chan LogEntry, logQueueSize), newEntriesAlert: NewAlert()}
}

func (env *TEnv) WithLock(f func (*TEnv)) {
    env.m.Lock()
    defer env.m.Unlock()
    f(env)
}

func (env *TEnv) CommitChanges(leaderCommit uint64) {
    if env.commitIndex < leaderCommit {
        maxIdx := leaderCommit
        if uint64(len(env.l.Entries)) < maxIdx {
            maxIdx = uint64(len(env.l.Entries) - 1)
        }

        entiresToCommit := env.l.Entries[env.lastApplied + 1 : maxIdx + 1]
        env.lastApplied = maxIdx
        env.commitIndex = leaderCommit
        for _, entry := range entiresToCommit {
            env.commitQueue <- entry
        }
    } else {
        maxIdx := env.commitIndex 
        if uint64(len(env.l.Entries)) < maxIdx {
            maxIdx = uint64(len(env.l.Entries) - 1)
        }

        entiresToCommit := env.l.Entries[env.lastApplied + 1 : maxIdx + 1]
        env.lastApplied = maxIdx
        for _, entry := range entiresToCommit {
            env.commitQueue <- entry
        }
    }
}

type ApplyRequest struct {
    op int
    key string
    value string
    resp chan bool
}

func (env *TEnv) ApplyRequestSync(op int, key string, value string) bool {
    statusChan := make(chan bool, 1)
    env.WithLock(func(env *TEnv) {
        env.l = Append(env.l, LogEntry{Op: op, Key: key, Value: value, statusChan: &statusChan,})
    })
    env.newEntriesAlert.Signal()
    return <-statusChan
}

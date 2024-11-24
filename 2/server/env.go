package main

import (
    "sync"
    "time"
)

type LeaderState struct {
    NextIndex []uint64
    MatchIndex []uint64
}

func NewLeaderState(numNodes int, lastLogIndex uint64) *LeaderState {
    nextIndex := make([]uint64, numNodes)
    for i := range nextIndex {
        nextIndex[i] = lastLogIndex + 1
    }

    return &LeaderState {
        NextIndex: nextIndex,
        MatchIndex: make([]uint64, numNodes),
    }
}

type TEnv struct {
    p PState
    l Log
    commitIndex uint64
    lastApplied uint64
    leaderState *LeaderState
    leaderId *uint64
    lastHB time.Time
    m sync.Mutex
}

func (env *TEnv) WithLock(f func (*TEnv)) {
    env.m.Lock()
    defer env.m.Unlock()
    f(env)
}

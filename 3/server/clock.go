package main

import (
    "sync"
)

type TTimestamp struct {
    Time uint64 `json:"time"`
    Id int `json:"id"`
}

type TClock struct {
    Ts TTimestamp
    M sync.Mutex
}

func NewClock(id int) *TClock {
    return &TClock{
        Ts: TTimestamp {
            Time: 0,
            Id: id,
        },
    }
}

func Compare(a TTimestamp, b TTimestamp) int {
    if a.Time != b.Time {
        if a.Time < b.Time {
            return -1
        } else {
            return 1
        }
    }

    if a.Id < b.Id {
        return -1
    } else if a.Id == b.Id {
        return 0
    } else {
        return 1
    }
}

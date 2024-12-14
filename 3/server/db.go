package main

import (
    "log"
    "sync"
)

type TVal struct {
    Val string  `json:"val"`
    Ts TTimestamp `json:"ts"`
    Deleted bool `json:"deleted"`
}

type TDb struct {
    Storage map[string]TVal
    M sync.Mutex
}

func NewDb() *TDb {
    return &TDb {
        Storage: make(map[string]TVal),
    }
}

func (db *TDb) Put(key string, value string, ts TTimestamp) {
    db.M.Lock()
    defer db.M.Unlock()

    if val, ok := db.Storage[key]; ok {
        cmp := Compare(val.Ts, ts)
        if cmp == 0 {
            log.Fatal("Timestamps can not be same")
        } else if cmp < 0 {
            db.Storage[key] = TVal{value, ts, false}
        }
    } else {
        db.Storage[key] = TVal{value, ts, false}
    }
}

func (db *TDb) Delete(key string, ts TTimestamp) {
    db.M.Lock()
    defer db.M.Unlock()
    if val, ok := db.Storage[key]; ok {
        cmp := Compare(val.Ts, ts)
        if cmp == 0 {
            log.Fatal("Timestamps can not be same")
        } else if cmp < 0 {
            db.Storage[key] = TVal{"", ts, true}
        }
    } else {
        db.Storage[key] = TVal{"", ts, true}
    }
}

func (db *TDb) Get(key string) (string, bool) {
    db.M.Lock()
    defer db.M.Unlock()
    if key, ok := db.Storage[key]; ok {
        return key.Val, !key.Deleted
    } else {
        return "", false
    }
}

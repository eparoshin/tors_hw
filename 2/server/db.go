package main

import (
    "sync"
    "strings"
    "context"
    "log"
)

type Db struct {
    data map[string]string
    m sync.RWMutex
}

func periodicUpdate(db *Db, ctx context.Context, commitQueue <- chan LogEntry) {
    log.Print("periodic update started")
    for {
        select {
        case <- ctx.Done():
            log.Print("quit periodic update")
            return
        case entry := <- commitQueue:
            log.Print("got new commit entry ", entry)
            db.CommitEntry(entry)
        }
    }
}

func NewDb(ctx context.Context, commitQueue <- chan LogEntry) *Db {
    db := Db{data: make(map[string]string),}
    go periodicUpdate(&db, ctx, commitQueue)

    return &db
}

func (db *Db) CommitEntry(entry LogEntry) {
    switch entry.Op {
    case CREATE:
        db.Create(entry.Key, entry.Value)
    case UPDATE:
        db.Update(entry.Key, entry.Value)
    case DELETE:
        db.Delete(entry.Key)
    }
}

func (db *Db) Get(key string) (string, bool) {
    db.m.RLock()
    defer db.m.RUnlock()

    val, ok := db.data[key]
    return strings.Clone(val), ok
}

func (db *Db) Create(key string, value string) bool {
    db.m.Lock()
    defer db.m.Unlock()

    if _, ok := db.data[key]; ok {
        return false
    } else {
        db.data[key] = value
        return true
    }
}

func (db *Db) Update(key string, value string) bool {
    db.m.Lock()
    defer db.m.Unlock()

    if _, ok := db.data[key]; ok {
        db.data[key] = value
        return true
    } else {
        return false
    }
}

func (db *Db) Delete(key string) bool {
    db.m.Lock()
    defer db.m.Unlock()

    if _, ok := db.data[key]; ok {
        delete(db.data, key)
        return true
    } else {
        return false
    }
}

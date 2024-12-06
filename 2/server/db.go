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
            status := db.CommitEntry(entry)
            if entry.statusChan != nil {
                *entry.statusChan <- status
            }
        }
    }
}

func NewDb(ctx context.Context, commitQueue <- chan LogEntry) *Db {
    db := Db{data: make(map[string]string),}
    go periodicUpdate(&db, ctx, commitQueue)

    return &db
}

func (db *Db) CommitEntry(entry LogEntry) bool {
    switch entry.Op {
    case CREATE:
        return db.Create(entry.Key, entry.Value)
    case UPDATE:
        return db.Update(entry.Key, entry.Value)
    case DELETE:
        return db.Delete(entry.Key)
    default:
        log.Fatalln("incorrect Op")
    }
    return false
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

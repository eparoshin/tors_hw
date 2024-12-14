package main

import (
    "os"
    "log"
    "fmt"
    "context"
)

func main() {
    configFile := os.Args[1]
    config, err := LoadConfigFromFile(configFile)
    if err != nil {
        log.Fatal(err)
    }

    var id int
    fmt.Sscanf(os.Args[2], "%d", &id)

    db := NewDb()
    clock := NewClock(id)

    var port int
    fmt.Sscanf(config[id], "http://localhost:%d", &port)

    server, err := NewServer(context.Background(), config, port, db, clock)
    if err != nil {
        log.Fatal(err)
    }

    log.Fatal(server.ListenAndServe());
}

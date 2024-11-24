package main

import (
    "log"
    "path/filepath"
    "flag"
    "context"
)

var Flags struct {
    NodeId uint64
    Workdir string
    NodesConfig string
    AppConfig string
}

func init() {
    flag.Uint64Var(&Flags.NodeId, "nodeid", 0, "")
    flag.StringVar(&Flags.Workdir, "workdir", "", "")
    flag.StringVar(&Flags.NodesConfig, "nodes-config", "", "")
    flag.StringVar(&Flags.AppConfig, "app-config", "", "")
}

func ParseFlags() {
    flag.Parse()
    if Flags.NodeId == 0 || Flags.Workdir == "" || Flags.NodesConfig == "" || Flags.AppConfig == "" {
        log.Fatal("Flags not set", Flags)
    }
}

func main() {
    ParseFlags()
    log.Print("Launching Node with flags: ", Flags)

    nodesConfig, err := NewNodesConfig(Flags.NodesConfig)
    if err != nil {
        log.Fatal("Error while reading nodes config: ", err)
    }

    appConfig, err := NewAppConfig(Flags.AppConfig)
    if err != nil {
        log.Fatal("Error while reading app config: ", err)
    }

    pState, err := NewPState(filepath.Join(Flags.Workdir, "pstate.json"))
    if err != nil {
        log.Fatal("Error while reading pstate: ", err)
    }

    raftLog, err := NewLog(filepath.Join(Flags.Workdir, "log.json"))
    if err != nil {
        log.Fatal("Error while reading log: ", err)
    }

    env := TEnv{p: pState, l: raftLog,}

    ctx := context.Background()

    raftServer, err := NewRaftServer(&env, ctx, nodesConfig, Flags.NodeId, appConfig)

    if err != nil {
        log.Fatal("Error while creating raft server: ", err)
    }

    log.Fatal(raftServer.ListenAndServe())

}

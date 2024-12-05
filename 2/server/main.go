package main

import (
    "log"
    "path/filepath"
    "flag"
    "context"
)

var Flags struct {
    NodeId int64
    Workdir string
    NodesConfig string
    AppConfig string
}

func init() {
    flag.Int64Var(&Flags.NodeId, "nodeid", -1, "")
    flag.StringVar(&Flags.Workdir, "workdir", "", "")
    flag.StringVar(&Flags.NodesConfig, "nodes-config", "", "")
    flag.StringVar(&Flags.AppConfig, "app-config", "", "")
}

func ParseFlags() {
    flag.Parse()
    if Flags.NodeId == -1 || Flags.Workdir == "" || Flags.NodesConfig == "" || Flags.AppConfig == "" {
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

    env := NewEnv(pState, raftLog, 100)

    ctx := context.Background()

    //db := NewDb(ctx, env.commitQueue)

    raftServer, err := NewRaftServer(&env, ctx, nodesConfig, uint64(Flags.NodeId), appConfig)

    if err != nil {
        log.Fatal("Error while creating raft server: ", err)
    }

    log.Fatal(raftServer.ListenAndServe())

}

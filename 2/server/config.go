package main

import (
    "os"
    "encoding/json"
    "fmt"
)

type NodeConfig struct {
    Host string `json:"host"`
    InternalPort int `json:"internal_port"`
    ExternalPort int `json:"external_port"`
}

func (node NodeConfig) InternalUri() string {
    return fmt.Sprintf("http://%s:%d", node.Host, node.InternalPort)
}

func (node NodeConfig) ExternalUri() string {
    return fmt.Sprintf("http://%s:%d", node.Host, node.ExternalPort)
}

type NodesConfig []NodeConfig

func NewNodesConfig(fileName string) (config NodesConfig, err error) {
    data, err := os.ReadFile(fileName)
    if err != nil {
        return
    }
    err = json.Unmarshal(data, &config)
    return
}

type AppConfig struct {
    HBTimeout int `json:"hb_timeout_ms"`
    RandomShift int `json:"random_shift_ms"`
    VoteRequestTimeoutMs int `json:"vote_request_timeout_ms"`
    AppendEntriesTimeoutMs int `json:"append_entries_timeout_ms"`
    HBIntervalMs int `json:"hb_interval_ms"`
}

func NewAppConfig(fileName string) (config AppConfig, err error) {
    data, err := os.ReadFile(fileName)
    if err != nil {
        return
    }
    err = json.Unmarshal(data, &config)
    return
}

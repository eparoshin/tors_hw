package main

import (
    "os"
    "encoding/json"
)

type NodeConfig struct {
    Host string `json:"host"`
    InternalPort int `json:"internal_port"`
    ExternalPort int `json:"external_port"`
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
}

func NewAppConfig(fileName string) (config AppConfig, err error) {
    data, err := os.ReadFile(fileName)
    if err != nil {
        return
    }
    err = json.Unmarshal(data, &config)
    return
}

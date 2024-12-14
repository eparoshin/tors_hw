package main

import (
    "os"
    "strings"
)

type TConfig []string

func LoadConfigFromFile(filename string) (TConfig, error) {
    data, err := os.ReadFile(filename)
	if err != nil {
        return nil, err
	}
    return LoadConfigFromString(string(data)), nil
}

func LoadConfigFromString(data string) TConfig {
    return strings.Split(data, "\n")
}

package main

import (
    "os"
    "encoding/json"
    "log"
    "errors"
    "io/fs"
)

type PState struct {
    State struct {
        CurrentTerm uint64 `json:"current_term"`
        VotedFor *uint64 `json:"voted_for"`
    }
    FileName string
}

func NewPState(fileName string) (state PState, err error) {
    state.FileName = fileName
    data, err := os.ReadFile(fileName)
    if err != nil {
        if errors.Is(err, fs.ErrNotExist) {
            if err = state.DumpPState(); err != nil {
                return
            }
        }
        return
    }

    err = json.Unmarshal(data, &state.State)
    return
}

func (state *PState)SetCurrentTerm (curr uint64) {
    state.State.CurrentTerm = curr
    if err := state.DumpPState(); err != nil {
        log.Fatal(err)
    }
}

func (state *PState)ResetVote () {
    state.State.VotedFor = nil
    if err := state.DumpPState(); err != nil {
        log.Fatal(err)
    }
}

func (state *PState)SetVote (vote uint64) {
    state.State.VotedFor = &vote
    if err := state.DumpPState(); err != nil {
        log.Fatal(err)
    }
}

func (state PState) DumpPState() error {
    name, err := func() (string, error) {
        file, err := os.CreateTemp("", "*")
        if err != nil {
            return "", err
        }
        name := file.Name()
        defer file.Close()
        data, err := json.Marshal(state.State)
        if err != nil {
            return name, err
        }

        _, err = file.Write(data)
        return name, err
    }()

    if err != nil {
        return err
    }

    return os.Rename(name, state.FileName)
}

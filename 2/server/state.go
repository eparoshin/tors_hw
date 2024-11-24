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
    data, err := os.ReadFile(fileName)
    if err != nil {
        if errors.Is(err, fs.ErrNotExist) {
            file, err := os.Create(fileName)
            if err == nil {
                defer file.Close()
            }
            log.Print("Empty PState file created")
        }
        return
    }

    state.FileName = fileName
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
        name := file.Name()
        if err != nil {
            return name, err
        }
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

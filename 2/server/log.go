package main

import (
    "io"
    "os"
    "encoding/json"
    "encoding/binary"
    "errors"
    "log"
)

const (
    CREATE = iota
    UPDATE
    DELETE
)

type LogEntry struct {
    Term uint64 `json:"term"`
    Op int `json:"op"`
    Key string `json:"key"`
    Value string `json:"value"`
}

type Log struct {
    FilePath string
    Entries []LogEntry
}

var EntryCorrupted = errors.New("Entry corrupted")

func SerializeEntry(entry LogEntry, writer io.Writer) (int, error) {
    data, err := json.Marshal(entry)
    if err != nil {
        return 0, err
    }
    lenData := make([]byte, 4)
    entryLen := len(data)
    binary.LittleEndian.PutUint32(lenData, uint32(entryLen))
    n, err := writer.Write(lenData)
    if err != nil {
        return n, err
    }

    m, err := writer.Write(data)

    return n + m, err
}

func DeserializeEntry(reader io.Reader, entry *LogEntry) (int, error) {
    lenData := make([]byte, 4)
    n, err := reader.Read(lenData)
    if err != nil {
        return n, err
    }

    if n != 4 {
        return n, EntryCorrupted
    }

    entryLen := binary.LittleEndian.Uint32(lenData)

    data := make([]byte, entryLen)
    m, err := reader.Read(data)
    if err != nil {
        return n + m, err
    }

    if m != int(entryLen) {
        return n + m, EntryCorrupted
    }

    err = json.Unmarshal(data, entry)

    return n + m, err
}

func NewLog(filePath string) (Log, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return Log{FilePath: filePath,}, err
    }

    defer file.Close()

    var entries []LogEntry
    offset := 0
    for {
        var entry LogEntry
        n, err := DeserializeEntry(file, &entry)
        if n == 0 && errors.Is(err, io.EOF) {
            break;
        } else if errors.Is(err, EntryCorrupted) {
            file.Truncate(int64(offset))
            log.Printf("File %s is corrupted, truncated at offset %d, entries length is %d", filePath, offset, len(entries))
            break
        } else if err != nil {
            return Log{}, err
        }

        offset += n
        entries = append(entries, entry)
    }

    return Log{filePath, entries}, nil
}

func Append(wlog Log, logEntry LogEntry) Log {
    file, err := os.OpenFile(wlog.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    if _, err := SerializeEntry(logEntry, file); err != nil {
        log.Fatal(err)
    }

    wlog.Entries = append(wlog.Entries, logEntry)
    return wlog;
}

func (wlog Log) Back() LogEntry {
    return wlog.Entries[len(wlog.Entries) - 1]
}

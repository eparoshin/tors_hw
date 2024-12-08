package main

import (
    "fmt"
    "encoding/json"
    "io"
    "log"
    "net/http"
    "bufio"
    "strings"
    "math/rand"
    "bytes"
    "os"
    "time"
    "context"
)

type NodeConfig struct {
    Host string `json:"host"`
    InternalPort int `json:"internal_port"`
    ExternalPort int `json:"external_port"`
}

func (node NodeConfig) ExternalUri() string {
    return fmt.Sprintf("http://%s:%d", node.Host, node.ExternalPort)
}

type NodesConfig []NodeConfig

var nodes NodesConfig

var client *http.Client

var timeout time.Duration

func getNodeId(nodeId int) int {
    if nodeId < 0 {
        return rand.Intn(len(nodes))
    } else {
        return nodeId
    }
}

func ProcessCreate(ctx context.Context, key string, value string, nodeId int) string {
    for {
        nodeId = getNodeId(nodeId)
        data, err := json.Marshal(map[string]string{"key": key, "value": value})
        if err != nil {
            log.Fatal(err)
        }
        req, err := http.NewRequestWithContext(ctx, "POST", nodes[nodeId].ExternalUri() + "/entry", bytes.NewReader(data))
        if err != nil {
            log.Fatal(err)
        }
        log.Print(req.URL)
        resp, err := client.Do(req)
        if err != nil {
            log.Println(err)
            nodeId = -1
            continue
        }

        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
            log.Print(err)
            nodeId = -1
            continue
        }
        resp.Body.Close()

        return fmt.Sprintf("resp: %+v\t body: %s", resp, string(respBody))
    }
}

func ProcessRead(ctx context.Context, key string, nodeId int) string {
    for {
        nodeId = getNodeId(nodeId)
        req, err := http.NewRequestWithContext(ctx, "GET", nodes[nodeId].ExternalUri() + "/entry/" + key, nil)
        if err != nil {
            log.Fatal(err)
        }
        log.Print(req.URL)
        resp, err := client.Do(req)
        if err != nil {
            log.Println(err)
            nodeId = -1
            continue
        }

        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
            log.Print(err)
            nodeId = -1
            continue
        }
        resp.Body.Close()

        return fmt.Sprintf("resp: %+v\t body: %s", resp, string(respBody))
    }
}

func ProcessUpdate(ctx context.Context, key string, value string, nodeId int) string {
    for {
        nodeId = getNodeId(nodeId)
        data, err := json.Marshal(map[string]string{"value": value})
        if err != nil {
            log.Fatal(err)
        }
        req, err := http.NewRequestWithContext(ctx, "PUT", nodes[nodeId].ExternalUri() + "/entry/" + key, bytes.NewReader(data))
        if err != nil {
            log.Fatal(err)
        }
        log.Print(req.URL)
        resp, err := client.Do(req)
        if err != nil {
            log.Println(err)
            nodeId = -1
            continue
        }

        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
            log.Print(err)
            nodeId = -1
            continue
        }
        resp.Body.Close()

        return fmt.Sprintf("resp: %+v\t body: %s", resp, string(respBody))
    }
}

func ProcessDelete(ctx context.Context, key string, nodeId int) string {
    for {
        nodeId = getNodeId(nodeId)
        req, err := http.NewRequestWithContext(ctx, "DELETE", nodes[nodeId].ExternalUri() + "/entry/" + key, nil)
        if err != nil {
            log.Fatal(err)
        }
        log.Print(req.URL)
        resp, err := client.Do(req)
        if err != nil {
            log.Println(err)
            nodeId = -1
            continue
        }

        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
            log.Print(err)
            nodeId = -1
            continue
        }
        resp.Body.Close()

        return fmt.Sprintf("resp: %+v\t body: %s", resp, string(respBody))
    }
}

func ProcessCas(ctx context.Context, key string, prevVal string, newVal string, nodeId int) string {
    for {
        nodeId = getNodeId(nodeId)
        data, err := json.Marshal(map[string]string{"prev_value": prevVal, "value": newVal})
        if err != nil {
            log.Fatal(err)
        }
        req, err := http.NewRequestWithContext(ctx, "PUT", nodes[nodeId].ExternalUri() + "/entry/" + key, bytes.NewReader(data))
        if err != nil {
            log.Fatal(err)
        }
        log.Print(req.URL)
        resp, err := client.Do(req)
        if err != nil {
            log.Println(err)
            nodeId = -1
            continue
        }

        respBody, err := io.ReadAll(resp.Body)
        if err != nil {
            log.Print(err)
            nodeId = -1
            continue
        }
        resp.Body.Close()

        return fmt.Sprintf("resp: %+v\t body: %s", resp, string(respBody))
    }
}

func Process(line string) string {
    lines := strings.Fields(strings.ToLower(line))
    log.Printf("Len lines: %d\n", len(lines))
    nodeId := -1
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    switch lines[0] {
    case "c":
        if len(lines) > 3 {
            fmt.Sscanf(lines[3], "%d", &nodeId)
        }
        return ProcessCreate(ctx, lines[1], lines[2], nodeId)
    case "r":
        if len(lines) > 2 {
            fmt.Sscanf(lines[2], "%d", &nodeId)
        }
        return ProcessRead(ctx, lines[1], nodeId)
    case "u":
        if len(lines) > 3 {
            fmt.Sscanf(lines[3], "%d", &nodeId)
        }
        return ProcessUpdate(ctx, lines[1], lines[2], nodeId)
    case "d":
        if len(lines) > 2 {
            fmt.Sscanf(lines[2], "%d", &nodeId)
        }
        return ProcessDelete(ctx, lines[1], nodeId)
    case "cas":
        if len(lines) > 4 {
            fmt.Sscanf(lines[4], "%d", &nodeId)
        }
        return ProcessCas(ctx, lines[1], lines[2], lines[3], nodeId)
    default:
        log.Fatalln(line)
    }
    return "UNREACHABLE"
}

func NewNodesConfig(fileName string) (config NodesConfig, err error) {
    data, err := os.ReadFile(fileName)
    if err != nil {
        return
    }
    err = json.Unmarshal(data, &config)
    return
}

func main() {
    client = &http.Client {
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            log.Println("Redirect", *req)
            for _, r := range via {
                log.Println("via", *r)
            }

            return nil
        },
    }
    configFile := os.Args[1]
    var err error
    nodes, err = NewNodesConfig(configFile)
    if err != nil {
        log.Fatal(err)
    }

    timeout = time.Second * 10

    stdin := bufio.NewReader(os.Stdin)
    for {
        line, err := stdin.ReadString('\n')
        if err != nil {
            log.Fatal(err)
        }
        fmt.Println(Process(line))
    }

}

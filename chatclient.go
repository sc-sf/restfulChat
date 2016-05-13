package main

import (
    "log"
    "bufio"
    "fmt"
    "os"
    "time"
    "bytes"
    "encoding/json"
    "net/http"
    "flag"
)

func join() {
    joinURL := "http://localhost:8080/chat/join/" + userName
    res, _ := http.Get(joinURL)  
    defer res.Body.Close()
}
type Msg struct {
    By       string  `json:"by"`
    To       string  `json:"to"`
    Text         string  `json:"text"`
    CreatedOn    string  `json:"createdon"`
}
func (m *Msg) write() {
    url := "http://" + serveraddress + "/chat/msgtoserver"
    j, _ := json.Marshal(m)
    buf := bytes.NewBuffer(j)
    resp, _ := http.Post(url, "application/json", buf)
    defer resp.Body.Close()

}
func (m *Msg) get() {
    url := "http://" + serveraddress + "/chat/msgtoclient/" + userName
    j, _ := json.Marshal(m)
    buf := bytes.NewBuffer(j)
    resp, _ := http.Post(url, "application/json; charset=utf-8", buf)
    defer resp.Body.Close()
    json.NewDecoder(resp.Body).Decode(m)
    fmt.Println(m.Text)
}
func (m *Msg) checkAndGetMessage() {
    url := "http://" + serveraddress + "/chat/checkmessage/" + userName
    res, _ := http.Get(url)
    if res.StatusCode == 202 {   // StatusAccepted = 202: Means there's message available
        m.get()
    }
    defer res.Body.Close()
}
var serveraddress string
var userName string

func main() {  
    flag.StringVar(&userName, "fname", "foo", "Type in your first name: e.g. fname=David")
    flag.StringVar(&serveraddress, "server", "localhost:8080", "Type in the address and " + 
        "port of the server this client connects to.")
    flag.Parse()
    join() 
    m := Msg{}
    ticker := time.NewTicker(time.Microsecond * 500)
    go func() {
        for _ = range ticker.C {
            m.checkAndGetMessage()    
        }
    }()
    reader := bufio.NewReader(os.Stdin)
    log.Println("\tConnected to server: " + serveraddress)
    for {
        txt, _ := reader.ReadString('\n') // ignoring error handling here
        if txt != "\n" {
            msg := Msg{ By: userName, CreatedOn: time.Now().String(), Text: txt, }
            go msg.write()
        }
        fmt.Printf("%v: ", userName)
    }
}
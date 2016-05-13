package main

import (
    "encoding/json"
    "log"
    "net/http"
    "sync"
    "flag"
    "github.com/gorilla/mux"
)

var (
    mStore *messageStore = &messageStore{ clients: make(map[string][]string) } 
    chatRoom = NewChatRoom()
    mu  sync.Mutex
    serveraddr  string
)

type messageStore struct {
    clients     map[string][]string
}

type Msg struct {
    By       string  `json:"by"`
    To       string  `json:"to"`
    Text         string  `json:"text"`
    CreatedOn    string  `json:"createdon"`
}

func (msg *Msg) SendTo(client string) {
    mu.Lock()
    defer mu.Unlock()    
    mStore.clients[client] = append(mStore.clients[client], msg.Text)
}

type ChatRoom struct {
    clients     map[string]bool
    //clients     []string
    incoming    chan Msg
    join        chan string
}

func (chatRoom *ChatRoom) Broadcast(m Msg) {
    chatRoom.clients[m.By] = false
    for client, notSender := range chatRoom.clients {
        if notSender {
            go m.SendTo(client)
        }    
    }
    chatRoom.clients[m.By] = true
}

func (chatRoom *ChatRoom) Listen() {
    go func() {
        for {
            select {
            case msg := <-chatRoom.incoming:
                chatRoom.Broadcast(msg)
            case user := <-chatRoom.join: 
                chatRoom.clients[user] = true
            }
        }
    }()
}

func NewChatRoom() *ChatRoom {
    chatRoom := &ChatRoom{
        clients: make(map[string]bool),
        join: make(chan string),
        incoming: make(chan Msg),
    }
    chatRoom.Listen()
    return chatRoom
}

func OutgoingMsgHandler(w http.ResponseWriter, r *http.Request) {
    getUser := mux.Vars(r)
    client := getUser["user"]
    msg := Msg{}
    mu.Lock()
    defer mu.Unlock()
    for _, m := range mStore.clients[client] {
        msg.Text = m
    }
    mStore.clients[client] = mStore.clients[client][1:] // removing the msg at index 0 after read
    j, err := json.Marshal(msg) 
    if err != nil { panic(err) }
    w.Write(j)
}

func IncomingMsgHandler(w http.ResponseWriter, r *http.Request) {
    msg := Msg{}
    err := json.NewDecoder(r.Body).Decode(&msg)
    if err != nil { panic(err) }
    msg.Text = "\n" + msg.By + " wrote: " + msg.Text 
    chatRoom.incoming <- msg
}

func JoinHandler(w http.ResponseWriter, r *http.Request) {
    getUser := mux.Vars(r)
    chatRoom.join <- getUser["user"]     
}

func CheckMsgHandler(w http.ResponseWriter, r *http.Request) {
    getUser := mux.Vars(r)
    user := getUser["user"]
    slice := mStore.clients[user]
    if ( len(slice) > 0 ) {
        w.WriteHeader(http.StatusAccepted) // Tell client to make another api call to get message
    }
}

func main() {
    flag.StringVar(&serveraddr, "server", "localhost:8080", 
        "Specifies the address and port this chat server listens on. \n\te.g. 127.0.0.1:8080")
    flag.Parse()
    r := mux.NewRouter().StrictSlash(false)   
    r.HandleFunc("/chat/msgtoserver", IncomingMsgHandler).Methods("POST")
    r.HandleFunc("/chat/msgtoclient/{user}", OutgoingMsgHandler).Methods("POST")
    r.HandleFunc("/chat/join/{user}", JoinHandler).Methods("GET")
    r.HandleFunc("/chat/checkmessage/{user}", CheckMsgHandler).Methods("GET")

    server := &http.Server{
        //Addr: "localhost:8080",
        Addr: serveraddr,
        Handler: r,
    }
    log.Println("\tListening on: " + serveraddr)
    server.ListenAndServe()
}
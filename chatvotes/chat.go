package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var wsConn *websocket.Conn

func closeWSConn() {
	if wsConn != nil {
		wsConn.Close()
	}
}

type message struct {
	Message string
}

// readFromChat connects to the chat server via websocket and reads messages.
// It looks for votes in the messages and sends them to the votes channel.
func readFromChat(votes chan<- string) {
	options, err := loadOptions()
	if err != nil {
		log.Fatalln("failed to load options:", err)
		return
	}

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/room"}
	log.Println("connecting to", u.String())

	// create websocket header with authentication if needed
	authData := map[string]any{
		"name":       "Anonymous",
		"avatar_url": "www.gravatar.com/avatar/0bc83cb571cd1c50ba6f3e8a78ef1346",
	}
	jsonBytes, err := json.Marshal(authData)
	if err != nil {
		log.Println("failed to marshal auth data:", err)
		return
	}
	authCookieValue := base64.StdEncoding.EncodeToString(jsonBytes)

	header := make(http.Header)
	header["cookie"] = []string{fmt.Sprintf("auth=%s", authCookieValue)}

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		log.Println("dial failed:", err)
		return
	}
	defer ws.Close()
	wsConn = ws
	log.Println("connected to", u.String())

	for {
		var msg message
		if err := ws.ReadJSON(&msg); err != nil {
			log.Println("error reading message:", err)
			break
		}
		for _, option := range options {
			if strings.Contains(
				strings.ToLower(msg.Message),
				strings.ToLower(option),
			) {
				log.Println("vote:", option)
				// send the vote to the votes channel
				votes <- option
			}
		}
	}
}

func startChatStream(stopchan <-chan struct{}, votes chan<- string) <-chan struct{} {
	// stoppedchan is closed when the chat reading goroutine stops.
	// 1 buffer to ensure no blocking on stop.
	stoppedchan := make(chan struct{}, 1)
	go func() {
		defer func() {
			stoppedchan <- struct{}{}
		}()
		for {
			select {
			// Check if we need to stop
			case <-stopchan:
				log.Println("stopping chat...")
				closeWSConn()
				return
			// Default case to run the chat reading
			default:
				log.Println("Connecting to chat...")
				readFromChat(votes)
				log.Println("Reconnecting to chat in 10 seconds...")
				time.Sleep(10 * time.Second) // wait before reconnecting
			}
		}
	}()
	return stoppedchan
}

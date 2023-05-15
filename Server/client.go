package main

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	connection    *websocket.Conn
	LastWriteTime int64
}

const CLIENT_WRITE_DELAY = time.Minute * 5

// TODO: detect disconnection
func (client *Client) Send(messageID int, message []byte) bool {
	err := client.connection.WriteMessage(messageID, message)
	if err != nil {
		log.Printf("Error sending message to client - %s", err.Error())
		return false
	}

	return true
}

func (client *Client) Run() {
	for {
		// client.connection.PI
	}
}

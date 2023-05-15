package main

import (
	"container/list"
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type ClientMessageService struct {
	clients        *list.List
	clientTable    map[*Client]*list.Element
	clientListLock sync.Mutex
}

type Message struct {
	X     int32 `json:"x"`
	Y     int32 `json:"y"`
	Color Color `json:"color"`
}

func NewClientMessageService() *ClientMessageService {
	return &ClientMessageService{
		clients:        list.New().Init(),
		clientTable:    make(map[*Client]*list.Element),
		clientListLock: sync.Mutex{}}
}

func (service *ClientMessageService) RegisterClient(client *Client) {
	service.clientListLock.Lock()
	defer service.clientListLock.Unlock()

	el := service.clients.PushBack(client)
	service.clientTable[client] = el
}

func (service *ClientMessageService) UnregisterClient(client *Client) {
	if client == nil {
		return
	}

	service.clientListLock.Lock()
	defer service.clientListLock.Unlock()

	el, ok := service.clientTable[client]
	if !ok || el == nil {
		return
	}

	service.clients.Remove(el)
}

func (service *ClientMessageService) BuildMessageFromPixel(pixel *Pixel) *Message {
	var x uint16 = uint16(pixel.Pos)
	var y uint16 = uint16(pixel.Pos >> 16)

	msg := Message{
		X:     int32(x),
		Y:     int32(y),
		Color: pixel.Col,
	}

	return &msg
}

func (service *ClientMessageService) Run(msgChannel <-chan *Pixel) {
	for {
		pixel := <-msgChannel

		msgStruct := service.BuildMessageFromPixel(pixel)
		msg, err := json.Marshal(msgStruct)
		if err != nil {
			log.Printf("[CMS] Error marshaling message, this message will be dropped - %s\n", err.Error())
			continue
		}

		service.clientListLock.Lock()
		for el := service.clients.Front(); el != nil; el = el.Next() {
			client := el.Value.(*Client)
			go client.Send(websocket.TextMessage, msg)
		}

		service.clientListLock.Unlock()
	}
}

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

type Color uint8

const COLOR_MASK = 0b0000_1111

const (
	WHITE       Color = iota
	BLACK       Color = iota
	BLUE        Color = iota
	GREEN       Color = iota
	RED         Color = iota
	ORANGE      Color = iota
	YELLOW      Color = iota
	BROWN       Color = iota
	PURPLE      Color = iota
	PINK        Color = iota
	LIGHTGREEN  Color = iota
	LIGHTBLUE   Color = iota
	LIGHTRED    Color = iota
	LIGHTYELLOW Color = iota
	MAROON      Color = iota
	VIOLET      Color = iota
)

var colorMap = map[Color]string{
	WHITE:       "#ffffff",
	BLACK:       "#000000",
	BLUE:        "#2450a4",
	GREEN:       "#00a368",
	RED:         "#be0039",
	ORANGE:      "#ffa800",
	YELLOW:      "#FFFF00",
	BROWN:       "#6d482f",
	PURPLE:      "#811e9f",
	PINK:        "#b44ac0",
	LIGHTGREEN:  "#7eed56",
	LIGHTBLUE:   "#3690ea",
	LIGHTRED:    "#be0049",
	LIGHTYELLOW: "ffd631",
	MAROON:      "#6d001a",
	VIOLET:      "#e4abff",
}

type Pixel struct {
	Pos  uint32 // x: uint16(Pos & uint16(1)), y: Pos >> 16
	Col  Color
	User string // owner's username
}

var g_WSConnUpgrader = websocket.Upgrader{CheckOrigin: CheckWSConnectionOrigin}
var g_clientMessageService *ClientMessageService = nil // initialized in main

func CheckWSConnectionOrigin(_ *http.Request) bool {
	return true
}

func HandleNewConnection(response http.ResponseWriter, request *http.Request) {
	ws, err := g_WSConnUpgrader.Upgrade(response, request, nil)
	if err != nil {
		log.Printf("Error upgrading client to websocket client - %s", err.Error())
		return
	}

	client := Client{connection: ws, LastWriteTime: 0}
	g_clientMessageService.RegisterClient(&client)
	go client.Run()
}

func HandleHealthCheck(response http.ResponseWriter, request *http.Request) {
	response.WriteHeader(http.StatusOK)
}

func init() {
	g_clientMessageService = NewClientMessageService()
}

func main() {
	http.HandleFunc("/ws", HandleNewConnection)
	http.HandleFunc("/healthcheck", HandleHealthCheck)

	updateChannel := make(chan *Pixel)
	Redis_Init(os.Getenv("REDIS_ENDPOINT"), os.Getenv("REDIS_PORT"), updateChannel)
	go g_clientMessageService.Run(updateChannel)

	err := http.ListenAndServe(":8000", nil)
	if err != nil {
		log.Fatalln("ListenAndServe: ", err.Error())
	}
}

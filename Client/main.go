package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"syscall/js"
)

type Board struct {
	Pixels []uint8 // each element will have 2 pixles (4 bits each)
	Width  uint16
	Height uint16
}

func Panicln(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func FetchBoard(_ js.Value, args []js.Value) interface{} {
	go func() {
		if len(args) != 2 {
			panic("Invalid # of args")
		}

		url := args[0]
		callback := args[1]

		res, err := http.Get(url.String())
		if err != nil || res.StatusCode != http.StatusOK {
			return
		}

		var board Board
		data, err := ioutil.ReadAll(res.Body)
		Panicln(err)

		err = json.Unmarshal(data, &board)
		Panicln(err)

		fmt.Println("Last Pixel Color: ", board.Pixels[len(board.Pixels)-1])

		callback.Invoke(string(board.Pixels), js.Null())
	}()
	return js.Null()
}

func main() {
	js.Global().Set("FetchBoard", js.FuncOf(FetchBoard))
	c := make(chan struct{})
	<-c
}

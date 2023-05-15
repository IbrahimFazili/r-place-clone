package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v9"
)

const BOARD_UPDATE_CHANNEL = "BoardUpdate"
const REDIS_CONNECTION_RETRIES = 3

var g_redisClient *redis.Client = nil

func Redis_InitClientInternal(addr string, port string) *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", addr, port)})

	success := false
	for i := 0; i < REDIS_CONNECTION_RETRIES; i++ {
		err := client.Ping(context.Background()).Err()
		if err != nil {
			log.Printf("[REDIS] Failed to ping redis node with addr (try %d) %s:%s - %s\n", i+1, addr, port, err.Error())
		} else {
			success = true
			break
		}

		time.Sleep(5 * time.Second)
	}

	if !success {
		log.Fatalln("[REDIS] Failed to establish connection to redis server, exiting")
		return nil
	}

	return client
}

func Redis_Init(addr string, port string, clientUpdateChannel chan<- *Pixel) {
	g_redisClient = Redis_InitClientInternal(addr, port)

	pubsub := g_redisClient.Subscribe(context.Background(), BOARD_UPDATE_CHANNEL)
	go Redis_WatchBoardUpdates(pubsub.Channel(), clientUpdateChannel)

	log.Println("[REDIS] Connected to redis")
}

func Redis_WatchBoardUpdates(messageChannel <-chan *redis.Message, clientUpdateChannel chan<- *Pixel) {
	for msg := range messageChannel {
		payload := []byte(msg.Payload)

		var pixel Pixel
		err := json.Unmarshal(payload, &pixel)
		if err != nil {
			log.Printf("[REDIS] Failed to deserailize pubsub message - %s\n", err.Error())
			continue
		}

		clientUpdateChannel <- &pixel
	}
}

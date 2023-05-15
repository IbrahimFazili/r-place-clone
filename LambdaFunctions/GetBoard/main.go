package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-redis/redis/v9"
)

const REDIS_PORT uint32 = 6379
const REDIS_CONNECTION_RETRIES = 3
const REDIS_BITFILED_KEY = "BoardBitfield"

// Could move these to redis if we need to change size of the board on the fly
const BOARD_WIDTH = 1000
const BOARD_HEIGHT = 1000

var g_redisClient *redis.Client = nil

type Board struct {
	Pixels []uint8 // each element will have 2 pixles (4 bits each)
	Width  uint16
	Height uint16
}

type ALBResponse events.ALBTargetGroupResponse

func Redis_InitClientInternal(addr string, port string) *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", addr, port)})

	err := client.Ping(context.Background()).Err()
	if err != nil {
		log.Printf("[REDIS] Failed to ping redis node with addr %s:%s - %s\n", addr, port, err.Error())
		log.Fatalln("[REDIS] Failed to establish connection to redis server, exiting")
		return nil
	}

	return client
}

func Redis_Init(addr string, port string) {
	g_redisClient = Redis_InitClientInternal(addr, port)
	log.Println("[REDIS] Connected to redis")
}

func Redis_ReadBoard(ctx context.Context) ([]uint8, error) {
	bitfield, err := g_redisClient.Get(ctx, REDIS_BITFILED_KEY).Result()
	if err != nil {
		log.Printf("[REDIS] Error reading board bitfield - %s\n", err.Error())
	}

	return []uint8(bitfield), err
}

func GetBase64EncodedBuffer(buffer []byte) []byte {
	length := base64.StdEncoding.EncodedLen(len(buffer))
	encodedBuffer := make([]byte, length)
	base64.StdEncoding.Encode(encodedBuffer, buffer)

	return encodedBuffer
}

func GetResponseHeaders() *map[string]string {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	return &headers
}

func GetErrorResponse() ALBResponse {
	return ALBResponse{StatusCode: http.StatusInternalServerError, StatusDescription: "500 Server Error", Headers: *GetResponseHeaders(), IsBase64Encoded: false}
}

func HandleRequest(ctx context.Context) (ALBResponse, error) {
	bitfield, err := Redis_ReadBoard(ctx)

	if err != nil {
		return GetErrorResponse(), err
	}

	body, err := json.Marshal(&Board{Pixels: bitfield, Width: BOARD_WIDTH, Height: BOARD_HEIGHT})
	if err != nil {
		return GetErrorResponse(), err
	}

	return ALBResponse{StatusCode: http.StatusOK, StatusDescription: "200 OK", Headers: *GetResponseHeaders(), Body: string(GetBase64EncodedBuffer(body)), IsBase64Encoded: true}, nil
}

func init() {
	Redis_Init(os.Getenv("REDIS_ENDPOINT"), os.Getenv("REDIS_PORT"))
}

func main() {
	lambda.Start(HandleRequest)
}

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-redis/redis/v9"
)

type ALBResponse events.ALBTargetGroupResponse
type ALBRequest events.ALBTargetGroupRequest

var g_redisClient *redis.Client = nil
var g_boardSize uint16 = 1000

type ReadRequest struct {
	User string
}

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

func GetBase64EncodedBuffer(buffer []byte) []byte {
	length := base64.StdEncoding.EncodedLen(len(buffer))
	encodedBuffer := make([]byte, length)
	base64.StdEncoding.Encode(encodedBuffer, buffer)

	return encodedBuffer
}

func GetBase64DecodedBuffer(buffer []byte) []byte {
	length := base64.StdEncoding.DecodedLen(len(buffer))
	decodedBuffer := make([]byte, length)
	base64.StdEncoding.Decode(decodedBuffer, buffer)

	return decodedBuffer
}

func GetResponseHeaders() *map[string]string {
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"

	return &headers
}

func GetResponse(statusCode int, statusDescription string) ALBResponse {
	return ALBResponse{StatusCode: statusCode, StatusDescription: statusDescription, Headers: *GetResponseHeaders(), IsBase64Encoded: false}
}

func GetWriteRequest(request ALBRequest) *ReadRequest {
	var rawRequest []byte = []byte(request.Body)
	if request.IsBase64Encoded {
		rawRequest = GetBase64DecodedBuffer([]byte(request.Body))
	}
	var writeRequest ReadRequest
	err := json.Unmarshal(rawRequest, &writeRequest)
	if err != nil {
		log.Println("Error in unmarshalling ALBRequest", err.Error())
		return nil
	}
	return &writeRequest
}

func HandleRequest(ctx context.Context, request ALBRequest) (ALBResponse, error) {
	event := GetWriteRequest(request)

	duration, err := g_redisClient.TTL(context.Background(), event.User).Result()
	if err != nil {
		log.Println("[REDIS]: Error getting ttl of user", err.Error())
		return GetResponse(http.StatusInternalServerError, "500 Internal Error"), errors.New("something wrong" + err.Error())
	}

	return ALBResponse{StatusCode: http.StatusOK, StatusDescription: "200 OK", Headers: *GetResponseHeaders(), Body: string(GetBase64EncodedBuffer([]byte(fmt.Sprintf("%f", duration.Seconds())))), IsBase64Encoded: true}, nil
}

func init() {
	Redis_Init(os.Getenv("REDIS_ENDPOINT"), os.Getenv("REDIS_PORT"))
}

func main() {
	lambda.Start(HandleRequest)
}

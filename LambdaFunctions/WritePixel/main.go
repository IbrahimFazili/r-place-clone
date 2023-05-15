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
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/go-redis/redis/v9"
	"github.com/gocql/gocql"
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

type Pixel struct {
	Pos  uint32 // x: uint16(Pos & uint16(1)), y: Pos >> 16
	Col  Color
	User string // owner's username
}

type WriteRequest struct {
	X    uint16
	Y    uint16
	Col  Color
	User string
}

type CassandraClient struct {
	Session *gocql.Session
	Config  *gocql.ClusterConfig
}

type ALBResponse events.ALBTargetGroupResponse
type ALBRequest events.ALBTargetGroupRequest

const REDIS_BITFILED_KEY = "BoardBitfield"
const BOARD_UPDATE_CHANNEL = "BoardUpdate"

var g_redisClient *redis.Client = nil
var g_cassndraClient *CassandraClient = nil
var g_boardSize uint16 = 1000

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

func validateInRange(i uint16, min uint16, max uint16) bool {
	if (i >= min) && (i <= max) {
		return true
	} else {
		return false
	}
}

func validateColor(color Color) bool {
	if color < WHITE || color > VIOLET {
		return false
	}
	return true
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

func GetWriteRequest(request ALBRequest) *WriteRequest {
	var rawRequest []byte = []byte(request.Body)
	if request.IsBase64Encoded {
		rawRequest = GetBase64DecodedBuffer([]byte(request.Body))
	}
	var writeRequest WriteRequest
	err := json.Unmarshal(rawRequest, &writeRequest)
	if err != nil {
		log.Println("Error in unmarshalling ALBRequest", err.Error())
		return nil
	}
	return &writeRequest
}

func Cassandra_Init() {
	cluster := gocql.NewCluster("cassandra.us-east-1.amazonaws.com")
	cluster.Port = 9142
	// add your service specific credentials
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: os.Getenv("AUTHENTICATION_USERNAME"),
		Password: os.Getenv("AUTHENTICATION_PASSWORD"),
	}
	// provide the path to the sf-class2-root.crt
	cluster.SslOpts = &gocql.SslOptions{
		CaPath:                 "./sf-class2-root.crt",
		EnableHostVerification: false,
	}

	// Override default Consistency to LocalQuorum
	cluster.Consistency = gocql.LocalQuorum
	cluster.DisableInitialHostLookup = false
	cluster.ProtoVersion = 4
	cluster.Keyspace = os.Getenv("KEYSPACE_NAME")
	cluster.ConnectTimeout = time.Second * 6

	session, err := cluster.CreateSession()
	if err != nil {
		fmt.Println("err>", err)
		return
	}

	g_cassndraClient = &CassandraClient{Session: session, Config: cluster}
}

func WriteToKeyspace(event WriteRequest) {
	if g_cassndraClient == nil {
		log.Println("[KEYSPACE]: cannot connect to Keyspace for writing")
		return
	}
	query_string := fmt.Sprintf("UPDATE %s.%s SET col=?, user=? WHERE pixel_x=? AND pixel_y=?", g_cassndraClient.Config.Keyspace, os.Getenv("KEYSPACE_TABLE"))
	err := g_cassndraClient.Session.Query(query_string, event.Col, event.User, event.X, event.Y).Exec()
	if err != nil {
		log.Println("[KEYSPACE]: error in adding to the table", err)
	}
}

func HandleRequest(ctx context.Context, request ALBRequest) (ALBResponse, error) {
	event := GetWriteRequest(request)
	// error handle for invalid requests
	if !validateInRange(event.X, 0, g_boardSize-1) || !validateInRange(event.Y, 0, g_boardSize-1) || !validateColor(event.Col) || len(event.User) == 0 {
		return GetResponse(http.StatusBadRequest, "400 Bad Request"), errors.New("invalid arguments")
	}

	// check in redis for 5 min window
	duration, err := g_redisClient.TTL(context.Background(), event.User).Result()
	if err != nil {
		log.Println("[REDIS]: Error getting ttl of user", err.Error())
	}
	if duration >= 0 {
		log.Printf("[REDIS]: Not enough time passed")
		return GetResponse(http.StatusNotAcceptable, "406 Not Acceptable"), errors.New("minimum time has not passed")
	}

	// write to redis
	_, e := g_redisClient.BitField(ctx, REDIS_BITFILED_KEY, "SET", "u4", fmt.Sprintf("#%d", uint32(event.X)+uint32(event.Y)*1000), fmt.Sprintf("%d", event.Col)).Result()
	if e != nil {
		log.Println("[REDIS]: Error setting in bitfield.", e.Error())
		return GetResponse(http.StatusInternalServerError, "500 Internal Error"), errors.New("error setting pixel in bitfield")
	}

	_, eu := g_redisClient.Set(ctx, event.User, "", 5*time.Minute).Result()
	if eu != nil {
		log.Println("[REDIS]: Error setting user", e.Error())
	}

	response := Pixel{
		Col:  event.Col,
		User: event.User,
		Pos:  (uint32(event.Y) << 16) | uint32(event.X),
	}

	serialized, err := json.Marshal(response)
	if err != nil {
		log.Println("[REDIS]: Error in marshalling Pixel", err)
		return GetResponse(http.StatusOK, "OK"), nil
	}

	g_redisClient.Publish(ctx, BOARD_UPDATE_CHANNEL, string(serialized)).Result()

	// write to cassandra
	WriteToKeyspace(*event)

	return GetResponse(http.StatusOK, "OK"), nil
}

func init() {
	Redis_Init(os.Getenv("REDIS_ENDPOINT"), os.Getenv("REDIS_PORT"))
	Cassandra_Init()
}

func main() {
	lambda.Start(HandleRequest)
}

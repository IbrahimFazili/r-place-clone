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

const REDIS_BITFILED_KEY = "BoardBitfield"
const BOARD_UPDATE_CHANNEL = "BoardUpdate"

var g_redisClient *redis.Client = nil

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
	X    uint16 `json:"pixel_x"`
	Y    uint16 `json:"pixel_y"`
	Col  Color  `json:"col"`
	User string `json:"user"`
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

var g_cassndraClient *CassandraClient = nil

type ALBResponse events.ALBTargetGroupResponse
type ALBRequest events.ALBTargetGroupRequest

var g_boardSize uint32 = 1000
var g_fetchedPixelArray []Pixel
var g_bitfield []uint8 = make([]uint8, (g_boardSize*g_boardSize)/2)

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
	cluster.ConnectTimeout = time.Second * 3

	session, err := cluster.CreateSession()
	if err != nil {
		fmt.Println("err>", err)
		return
	}

	g_cassndraClient = &CassandraClient{Session: session, Config: cluster}
}

func GetAllCassandraData() {
	if g_cassndraClient == nil {
		log.Println("[KEYSPACE]: cannot connect to Keyspace for reading")
	}
	query_string := fmt.Sprintf("SELECT * FROM %s.%s ALLOW FILTERING", g_cassndraClient.Config.Keyspace, os.Getenv("KEYSPACE_TABLE"))
	iter := g_cassndraClient.Session.Query(query_string).Iter().Scanner()
	for iter.Next() {
		var peer Pixel
		err := iter.Scan(&peer.X, &peer.Y, &peer.Col, &peer.User)
		if err != nil {
			log.Fatal(err)
		} else {
			g_fetchedPixelArray = append(g_fetchedPixelArray, peer)
		}
	}
}

func HandleCassandraData() {
	for _, pixel := range g_fetchedPixelArray {
		var x = pixel.X
		var y = pixel.Y
		var offset = (uint32(x) + g_boardSize*uint32(y)) / 2
		var col = pixel.Col
		if x%2 == 0 {
			// first 4 bits
			g_bitfield[offset] = uint8(col<<4) | (g_bitfield[offset])
		} else {
			// last 4 bits
			g_bitfield[offset] = (g_bitfield[offset]) | uint8(col&0x0F)
		}
	}
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

func HandleRequest(ctx context.Context, request ALBRequest) (ALBResponse, error) {
	// get data
	GetAllCassandraData()
	HandleCassandraData()

	// write to redis
	// _, e := g_redisClient.BitField(ctx, REDIS_BITFILED_KEY, "SET", "u4", fmt.Sprintf("#%d", 999999), fmt.Sprintf("%d", 0)).Result()
	_, e := g_redisClient.Set(ctx, REDIS_BITFILED_KEY, g_bitfield, 0).Result()
	if e != nil {
		log.Println("[REDIS]: Error setting in bitfield.", e.Error())
		return GetResponse(http.StatusInternalServerError, "500 Internal Error"), errors.New("error setting pixel in bitfield")
	}

	return GetResponse(http.StatusOK, "OK"), nil
}

func init() {
	Redis_Init(os.Getenv("REDIS_ENDPOINT"), os.Getenv("REDIS_PORT"))
	Cassandra_Init()
}

func main() {
	lambda.Start(HandleRequest)
}

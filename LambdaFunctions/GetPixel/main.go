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
	X    uint16 `json:"pixel_x"`
	Y    uint16 `json:"pixel_y"`
	Col  Color  `json:"col"`
	User string `json:"user"`
}

type ReadRequest struct {
	X uint16
	Y uint16
}

type ALBResponse events.ALBTargetGroupResponse
type ALBRequest events.ALBTargetGroupRequest

type CassandraClient struct {
	Session *gocql.Session
	Config  *gocql.ClusterConfig
}

var g_cassndraClient *CassandraClient = nil

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

func GetReadRequest(request ALBRequest) *ReadRequest {
	var rawRequest []byte = []byte(request.Body)
	if request.IsBase64Encoded {
		rawRequest = GetBase64DecodedBuffer([]byte(request.Body))
	}
	var readRequest ReadRequest
	err := json.Unmarshal(rawRequest, &readRequest)
	if err != nil {
		log.Println("Error in unmarshalling ALBRequest", err.Error())
		return nil
	}
	return &readRequest
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

func ReadFromKeyspace(ctx context.Context, event ReadRequest) (ALBResponse, error) {
	if g_cassndraClient == nil {
		log.Println("[KEYSPACE]: cannot connect to Keyspace for reading")
		return GetResponse(http.StatusInternalServerError, "500 Internal Error"), errors.New("cannot connect to Keyspace")
	}
	var fetchedPixel Pixel = Pixel{}
	log.Println("event", event)
	query_string := fmt.Sprintf("SELECT * FROM %s.%s WHERE pixel_x=? AND pixel_y=? ALLOW FILTERING", os.Getenv("KEYSPACE_NAME"), os.Getenv("KEYSPACE_TABLE"))
	err := g_cassndraClient.Session.Query(query_string, event.X, event.Y).WithContext(ctx).Scan(&fetchedPixel.X, &fetchedPixel.Y, &fetchedPixel.Col, &fetchedPixel.User)
	if err != nil {
		log.Println("[KEYSPACE]: error in reading from database", err)
		return GetResponse(http.StatusInternalServerError, "500 Internal Error"), errors.New("cannot find values in Keyspace")
	}
	serialized, serialerr := json.Marshal(fetchedPixel)
	if serialerr != nil {
		log.Println("[KEYSPACE]: error in marshalling pixel", serialerr)
		return GetResponse(http.StatusInternalServerError, "500 Internal Error"), errors.New("cannot marshall fetched pixel")
	}
	return ALBResponse{StatusCode: http.StatusOK, StatusDescription: "200 OK", Headers: *GetResponseHeaders(), Body: string(GetBase64EncodedBuffer(serialized)), IsBase64Encoded: true}, nil
}

func init() {
	Cassandra_Init()
}

func HandleRequest(ctx context.Context, request ALBRequest) (ALBResponse, error) {
	ev := GetReadRequest(request)
	return ReadFromKeyspace(ctx, *ev)
}

func main() {
	lambda.Start(HandleRequest)
}

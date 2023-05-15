#!/bin/bash
GOOS=linux GOARCH=amd64 go build .
zip GetBoard.zip GetBoard
aws s3 cp GetBoard.zip s3://a3-test

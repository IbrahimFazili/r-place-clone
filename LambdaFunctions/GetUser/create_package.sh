#!/bin/bash
GOOS=linux GOARCH=amd64 go build .
zip GetUser.zip GetUser
aws s3 cp GetUser.zip s3://a3-test

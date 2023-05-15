#!/bin/bash
GOOS=linux GOARCH=amd64 go build .
zip InitializeRedis.zip InitializeRedis sf-class2-root.crt
aws s3 cp InitializeRedis.zip s3://a3-test

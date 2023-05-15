#!/bin/bash
GOOS=linux GOARCH=amd64 go build .
zip WritePixel.zip WritePixel sf-class2-root.crt
aws s3 cp WritePixel.zip s3://a3-test

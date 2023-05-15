#!/bin/bash
GOOS=linux GOARCH=amd64 go build .
zip GetPixel.zip GetPixel sf-class2-root.crt
aws s3 cp GetPixel.zip s3://a3-test

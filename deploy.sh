#!/bin/bash
aws cloudformation deploy --capabilities CAPABILITY_NAMED_IAM --stack-name a3-test --template-file $1
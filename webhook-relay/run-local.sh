#!/bin/bash
export AWS_REGION=ap-northeast-1
export QUEUE_URL=https://sqs.ap-northeast-1.amazonaws.com/021930045083/github-webhook	
export JENKINS_BASE_URL=http://172.17.0.2:8080/
./bin/webhook -mode client

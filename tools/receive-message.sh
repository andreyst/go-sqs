#!/bin/bash
set -xe 

curl -s -X POST -d "Action=ReceiveMessage&QueueUrl=myqueue" -H "Content-Type: application/x-www-form-urlencoded" localhost:8080

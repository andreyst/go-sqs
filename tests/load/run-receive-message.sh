#!/bin/bash
set -xe

if [ -z "${CONNECTIONS}" ]; then
    CONNECTIONS=500
fi

if [ -z "${THREADS}" ]; then
    THREADS=500
fi

if [ -z "${DURATION}" ]; then
    DURATION=30
fi

if [ -z "${ENDPOINT}" ]; then
    ENDPOINT="http://localhost:8080"
fi

curl -s -X POST -d "Action=CreateQueue&QueueName=load_test_queue" -H "Content-Type: application/x-www-form-urlencoded" "${ENDPOINT}"
wrk -s receive-message.lua -c "${CONNECTIONS}" -t "${THREADS}" -d "${DURATION}" "${ENDPOINT}"

#!/bin/bash
set -xe

wrk -s wrk.lua -c 500 -t 100 -d 30 http://localhost:8080

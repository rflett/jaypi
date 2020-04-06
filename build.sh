#!/bin/bash

rm -rf ./bin
GOOS=linux go build -ldflags="-s -w" -o bin/getUser rest/user/getUser/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/updateUser rest/user/updateUser/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/updateGuesses rest/guesses/updateGuesses/main.go

#!/bin/bash

rm -rf ./bin
GOOS=linux go build -ldflags="-s -w" -o bin/createUser rest/user/createUser/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/getUser rest/user/getUser/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/updateUser rest/user/updateUser/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/createGroup rest/group/createGroup/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/getGroup rest/group/getGroup/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/updateGroup rest/group/updateGroup/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/joinGroup rest/group/joinGroup/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/leaveGroup rest/group/leaveGroup/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/createVote rest/votes/createVote/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/deleteVote rest/votes/deleteVote/main.go

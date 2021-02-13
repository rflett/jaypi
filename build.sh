#!/bin/bash

rm -rf ./bin
GOOS=linux go build -ldflags="-s -w" -o bin/createUser rest/user/createUser/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/getUser rest/user/getUser/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/updateUser rest/user/updateUser/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/createGroup rest/group/createGroup/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/getGroup rest/group/getGroup/main.go
GOOS=linux go build -ldflags="-s -w" -o bin/updateGroup rest/group/updateGroup/main.go
#GOOS=linux go build -ldflags="-s -w" -o bin/getSongs rest/guesses/getSongs/main.go
#GOOS=linux go build -ldflags="-s -w" -o bin/updateGuesses rest/guesses/updateGuesses/main.go
#GOOS=linux go build -ldflags="-s -w" -o bin/getGroupMembers rest/groupOld/getGroupMembers/main.go
#GOOS=linux go build -ldflags="-s -w" -o bin/addMember rest/groupOld/addMember/main.go
#GOOS=linux go build -ldflags="-s -w" -o bin/removeMember rest/groupOld/removeMember/main.go
#GOOS=linux go build -ldflags="-s -w" -o bin/createInvite rest/invites/createInvite/main.go
#GOOS=linux go build -ldflags="-s -w" -o bin/decodeInvite rest/invites/decodeInvite/main.go

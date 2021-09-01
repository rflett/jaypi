#!/bin/zsh

rm -rf ./bin/*

gofmt -s -w .

go build -ldflags="-s -w" -o bin/getUser rest/user/getUser/main.go
echo "Built getUser"
go build -ldflags="-s -w" -o bin/getUsersVotes rest/user/getUsersVotes/main.go
echo "Built getUsersVotes"
go build -ldflags="-s -w" -o bin/updateUser rest/user/updateUser/main.go
echo "Built updateUser"
go build -ldflags="-s -w" -o bin/getAvatarURL rest/user/getAvatarURL/main.go
echo "Built getAvatarURL"
go build -ldflags="-s -w" -o bin/registerDevice rest/device/registerDevice/main.go
echo "Built registerDevice"
go build -ldflags="-s -w" -o bin/deregisterDevice rest/device/deregisterDevice/main.go
echo "Built deregisterDevice"
go build -ldflags="-s -w" -o bin/createGroup rest/group/createGroup/main.go
echo "Built createGroup"
go build -ldflags="-s -w" -o bin/getGroup rest/group/getGroup/main.go
echo "Built getGroup"
go build -ldflags="-s -w" -o bin/deleteGroup rest/group/deleteGroup/main.go
echo "Built deleteGroup"
go build -ldflags="-s -w" -o bin/getGroupMembers rest/group/getGroupMembers/main.go
echo "Built getGroupMembers"
go build -ldflags="-s -w" -o bin/updateGroup rest/group/updateGroup/main.go
echo "Built updateGroup"
go build -ldflags="-s -w" -o bin/updateGroupOwner rest/user/updateGroupOwner/main.go
echo "Built updateGroupOwner"
go build -ldflags="-s -w" -o bin/joinGroup rest/group/joinGroup/main.go
echo "Built joinGroup"
go build -ldflags="-s -w" -o bin/leaveGroup rest/group/leaveGroup/main.go
echo "Built leaveGroup"
go build -ldflags="-s -w" -o bin/getGroupQR rest/group/getGroupQR/main.go
echo "Built getGroupQR"
go build -ldflags="-s -w" -o bin/createGame rest/group/createGame/main.go
echo "Built createGame"
go build -ldflags="-s -w" -o bin/deleteGame rest/group/deleteGame/main.go
echo "Built deleteGame"
go build -ldflags="-s -w" -o bin/updateGame rest/group/updateGame/main.go
echo "Built updateGame"
go build -ldflags="-s -w" -o bin/getGames rest/group/getGames/main.go
echo "Built getGames"
go build -ldflags="-s -w" -o bin/songSearch rest/song/songSearch/main.go
echo "Built songSearch"
go build -ldflags="-s -w" -o bin/getPlayedSongs rest/song/getPlayedSongs/main.go
echo "Built getPlayedSongs"
go build -ldflags="-s -w" -o bin/purgeSongs rest/song/purgeSongs/main.go
echo "Built purgeSongs"
go build -ldflags="-s -w" -o bin/createVote rest/votes/createVote/main.go
echo "Built createVote"
go build -ldflags="-s -w" -o bin/deleteVote rest/votes/deleteVote/main.go
echo "Built deleteVote"
go build -ldflags="-s -w" -o bin/signup rest/account/signup/main.go
echo "Built signup"
go build -ldflags="-s -w" -o bin/signin rest/account/signin/main.go
echo "Built signin"
go build -ldflags="-s -w" -o bin/validateJwt rest/account/validateJwt/main.go
echo "Built validateJwt"
go build -ldflags="-s -w" -o bin/oauthAuthenticate rest/oauth/authenticate/main.go
echo "Built oauthAuthenticate"
go build -ldflags="-s -w" -o bin/oauthCallback rest/oauth/callback/main.go
echo "Built oauthCallback"
go build -ldflags="-s -w" -o bin/chuneMachine lambda/chune-machine/main.go
echo "Built chuneMachine"
go build -ldflags="-s -w" -o bin/beanCounter lambda/bean-counter/main.go
echo "Built beanCounter"
go build -ldflags="-s -w" -o bin/scoreTaker lambda/score-taker/main.go
echo "Built scoreTaker"
go build -ldflags="-s -w" -o bin/authorizer lambda/authorizer/main.go
echo "Built authorizer"
go build -ldflags="-s -w" -o bin/townCrier lambda/town-crier/main.go
echo "Built townCrier"

echo "Done"

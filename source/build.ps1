Get-ChildItem -Path .\bin -Include * -File -Recurse | ForEach-Object { $_.Delete() }

$env:GOOS = "linux"

gofmt -s -w .

go build -ldflags="-s -w" -o bin/getUser rest/user/getUser/main.go
go build -ldflags="-s -w" -o bin/updateUser rest/user/updateUser/main.go
go build -ldflags="-s -w" -o bin/getAvatarURL rest/user/getAvatarURL/main.go

go build -ldflags="-s -w" -o bin/registerDevice rest/device/registerDevice/main.go
go build -ldflags="-s -w" -o bin/deregisterDevice rest/device/deregisterDevice/main.go

go build -ldflags="-s -w" -o bin/createGroup rest/group/createGroup/main.go
go build -ldflags="-s -w" -o bin/getGroup rest/group/getGroup/main.go
go build -ldflags="-s -w" -o bin/getGroupMembers rest/group/getGroupMembers/main.go
go build -ldflags="-s -w" -o bin/updateGroup rest/group/updateGroup/main.go
go build -ldflags="-s -w" -o bin/joinGroup rest/group/joinGroup/main.go
go build -ldflags="-s -w" -o bin/leaveGroup rest/group/leaveGroup/main.go
go build -ldflags="-s -w" -o bin/getGroupQR rest/group/getGroupQR/main.go
go build -ldflags="-s -w" -o bin/createGame rest/group/createGame/main.go
go build -ldflags="-s -w" -o bin/deleteGame rest/group/deleteGame/main.go
go build -ldflags="-s -w" -o bin/updateGame rest/group/updateGame/main.go
go build -ldflags="-s -w" -o bin/getGames rest/group/getGames/main.go

go build -ldflags="-s -w" -o bin/songSearch rest/song/songSearch/main.go
go build -ldflags="-s -w" -o bin/createVote rest/votes/createVote/main.go
go build -ldflags="-s -w" -o bin/deleteVote rest/votes/deleteVote/main.go

go build -ldflags="-s -w" -o bin/signup rest/account/signup/main.go
go build -ldflags="-s -w" -o bin/signin rest/account/signin/main.go
go build -ldflags="-s -w" -o bin/oauthAuthenticate rest/oauth/authenticate/main.go
go build -ldflags="-s -w" -o bin/oauthCallback rest/oauth/callback/main.go

go build -ldflags="-s -w" -o bin/chuneMachine lambda/chune-machine/main.go
go build -ldflags="-s -w" -o bin/beanCounter lambda/bean-counter/main.go
go build -ldflags="-s -w" -o bin/scoreTaker lambda/score-taker/main.go
go build -ldflags="-s -w" -o bin/authorizer lambda/authorizer/main.go
go build -ldflags="-s -w" -o bin/townCrier lambda/town-crier/main.go

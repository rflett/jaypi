Get-ChildItem -Path .\bin -Include * -File -Recurse | ForEach-Object { $_.Delete() }

$env:GOOS = "linux"

go build -ldflags="-s -w" -o bin/getUser rest/user/getUser/main.go
go build -ldflags="-s -w" -o bin/updateUser rest/user/updateUser/main.go

go build -ldflags="-s -w" -o bin/createGroup rest/group/createGroup/main.go
go build -ldflags="-s -w" -o bin/getGroup rest/group/getGroup/main.go
go build -ldflags="-s -w" -o bin/updateGroup rest/group/updateGroup/main.go
go build -ldflags="-s -w" -o bin/joinGroup rest/group/joinGroup/main.go
go build -ldflags="-s -w" -o bin/leaveGroup rest/group/leaveGroup/main.go
go build -ldflags="-s -w" -o bin/getGroupQR rest/group/getGroupQR/main.go

go build -ldflags="-s -w" -o bin/createVote rest/votes/createVote/main.go
go build -ldflags="-s -w" -o bin/deleteVote rest/votes/deleteVote/main.go

go build -ldflags="-s -w" -o bin/signup rest/account/signup/main.go
go build -ldflags="-s -w" -o bin/signin rest/account/signin/main.go
go build -ldflags="-s -w" -o bin/oauthAuthenticate rest/oauth/authenticate/main.go
go build -ldflags="-s -w" -o bin/oauthProviderRedirect rest/oauth/callback/main.go

go build -ldflags="-s -w" -o bin/chuneMachine lambda/chune-machine/main.go
go build -ldflags="-s -w" -o bin/beanCounter lambda/bean-counter/main.go
go build -ldflags="-s -w" -o bin/scoreTaker lambda/score-taker/main.go

# JAYPI (jay-pee-eye)

[![Go](https://github.com/rflett/jaypi/actions/workflows/main.yml/badge.svg?branch=main&event=push)](https://github.com/rflett/jaypi/actions/workflows/main.yml)

An API built on Golang for the soon to be JJJ countdown leaderboard app.

## Development

  - Go 1.14+
  - Serverless 2.25.2+
  - Docker

You'll also need AWS credentials.

Install the required Go modules with `go mod download`

## Running

Set up the required environment variables for [local invocation using a .env file](https://www.serverless.com/framework/docs/environment-variables/). 

## Deployment

### CI
Push to GitHub and the workflow will build and deploy on push to the `main` branch.


### Local
To deploy locally:

```bash
cd source
./build.ps1
cd ..
serverless deploy
```

If you just need to update the function code then you can quickly build and deploy by doing

```bash
cd source
go build -ldflags="-s -w" -o bin/deregisterDevice rest/device/deregisterDevice/main.go
cd ..
serverless deploy --function deregisterDevice --force
```

which only updates the function code and no other config.

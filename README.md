# JAYPI (jay-pee-eye)

[![Go Build](https://github.com/rflett/jaypi/actions/workflows/main.yml/badge.svg?branch=main&event=push)](https://github.com/rflett/jaypi/actions/workflows/main.yml)

An API built on Golang for the soon to be JJJ countdown leaderboard app.

## Query the live API

Use the Postman collection and create an environment setting the following variables:

- `host=https://api.staging.jaypi.online`

Postman tests will handle creating and updating the other variables as requests are made.

## Development

  - Go 1.14+
  - Serverless 2.25.2+

You'll also need AWS credentials.

Install the required Go modules with `go mod download` in the `source/` directory.

## Deployment

Push to GitHub and the workflow will build and deploy on push to the `main` branch or via a manual run.


---

### Doing things locally

Set up the required environment variables for [local invocation using a .env file](https://www.serverless.com/framework/docs/environment-variables/).

To deploy locally:

```bash
cd source
./build.sh
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

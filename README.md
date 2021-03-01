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

Under the `source/` directory build it with `./build.ps1` then start the function with its corresponding mock data under `mock/`.

```bash
# build the binaries used by serverless
cd source
./build.ps1

# invoke the function you want with its mock data
cd ..
serverless invoke local -f getUser --path mock/user/get.json
```

## Deployment

### CI
Push to GitHub and the workflow will build and deploy on push to the `main` branch.


### Local
To deploy locally, run the build script and then either

```
serverless deploy -f theFunctionName --force
```

to deploy a specific function (which is quick as it only updates the Lambda code) or

```
serverless deploy
```

which deploys the whole stack and takes forever.

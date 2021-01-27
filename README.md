# JAYPI (jay-pee-eye)

An API built on Golang for the soon to be JJJ countdown leaderboard app.

## Development

  - Go 1.14+
  - Serverless
  - Docker

You'll also need AWS credentials.

Install the required Go modules with `go mod download`

## Running

Build it first with `./build.sh` then start the function with its corresponding mock data under `mock/`.

```bash
# build the binaries used by serverless
./build.

# invoke the function you want with its mock data
serverless invoke local -f getUser --path mock/user/get.json
```

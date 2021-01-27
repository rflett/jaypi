# JJJ API

## Development

It's a serverless golang project, so at a minimum you'll need:

  - Go 1.13+
  - Serverless
  - Docker

You'll also need AWS credentials.

Install the required Go modules with `go mod download`

## Running

Build it first with `./build.sh` then start the function with its corresponding mock data under `mock/`.

```bash
# build the binaries used by serverless
./build.sh
# invoke the function you want with its mock data
serverless invoke local -f getUser --path mock/user/get.json
```

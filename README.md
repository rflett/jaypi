# JJJ API

## Development

It's a serverless golang project, so at a minimum you'll need:

  - Go 1.13+
  - Serverless
  - Docker

You'll also need AWS credentials.


## Running

Build it first with `./build.sh` then start the function with its corresponding mock data under `mock/`.

```bash
serverless invoke local -f getUser --path mock/user/get.json
```

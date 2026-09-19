

# Quick Start

```bash
# Initialize new certificates for user signing and nats server authentication
docker run --rm -it -v $(pwd)/scripts/:/scripts:ro -v $(pwd)/nsc:/nsc --entrypoint '/scripts/setup.sh' natsio/nats-box:latest

# Launch the docker compose stack
docker compose up --build

# Open https://localhost
```

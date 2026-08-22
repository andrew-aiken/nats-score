

# Quick Start

```bash
# Initialize new certificates for signing and authentication
docker run --rm -it -v $(pwd)/scripts/:/scripts:ro -v $(pwd)/nsc:/nsc --entrypoint '/scripts/setup.sh' natsio/nats-box:latest

# Launch the docker compose stack
docker compose up --build

# Open http://localhost:3000
```

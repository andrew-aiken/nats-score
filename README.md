
![License](https://img.shields.io/badge/License-GLP%203.0-blue.svg)
![GitHub Release](https://img.shields.io/github/v/release/andrew-aiken/nats-score)
![Tests](https://img.shields.io/github/actions/workflow/status/andrew-aiken/nats-score/gotest.yaml)


## Quick Start

```bash
# Pull the code locally
git clone https://github.com/andrew-aiken/nats-score.git

# Initialize new certificates for user signing and nats server authentication
docker run --rm -it -v $(pwd)/scripts/:/scripts:ro -v $(pwd)/nsc:/nsc --entrypoint '/scripts/setup.sh' natsio/nats-box:latest

# Generate caddy TLS certificates
bash ./caddy/certs/generate.sh

# Launch the docker compose stack
docker compose up --build

# Open https://localhost
```

## Contributing

### Testing

```bash
# Verify formatting is correct
gofmt -l .

# Lint
golangci-lint run

# Check for security findings, findings are allowed to be suppressed but need to be documented
gosec ./...

# Verify the tests all are functioning or just target a specific check being modified
go test -v -race ./...
```

#### Coverage

Checks should aim to have ~80% or more test coverage

```bash
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```


## Similar Projects
- [Quotient](https://github.com/dbaseqp/Quotient)
- [Scorestack](https://github.com/scorestack/scorestack)
- [Scorify](https://github.com/Scorify/Scorify)
- [Scoring Engine](https://github.com/scoringengine/scoringengine) <!-- cli commands & bash scripts for checks -->
- [Scoring-Engine (C2 Games)](https://gitlab.com/c2-games/scoring) <!-- Not going to lie, this one is all over the place -->

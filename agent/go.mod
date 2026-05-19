module github.com/andrew-aiken/nats-score/agent

go 1.26.2

require (
	github.com/andrew-aiken/checks v0.0.0-00010101000000-000000000000
	github.com/creasty/defaults v1.8.0
	github.com/nats-io/nats.go v1.47.0
	github.com/urfave/cli/v3 v3.6.1
)

replace github.com/andrew-aiken/checks => ../../checks

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/miekg/dns v1.1.69 // indirect
	github.com/nats-io/nkeys v0.4.11 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/prometheus-community/pro-bing v0.7.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
)

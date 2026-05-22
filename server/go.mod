module server

go 1.26

replace github.com/andrew-aiken/checks => ../../checks

require (
	github.com/andrew-aiken/checks v0.0.0-00010101000000-000000000000
	github.com/creasty/defaults v1.8.0
	github.com/go-co-op/gocron/v2 v2.19.0
	github.com/nats-io/jwt/v2 v2.8.1
	github.com/nats-io/nats.go v1.52.0
	github.com/nats-io/nkeys v0.4.15
	github.com/urfave/cli/v3 v3.8.0
	golang.org/x/oauth2 v0.36.0
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/miekg/dns v1.1.69 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/prometheus-community/pro-bing v0.7.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/net v0.53.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/tools v0.44.0 // indirect
)

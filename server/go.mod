module server

go 1.25.5

require golang.org/x/oauth2 v0.34.0

require (
	github.com/aaiken/nats-score/checks v0.0.0-00010101000000-000000000000
	github.com/creasty/defaults v1.8.0
	github.com/go-co-op/gocron/v2 v2.19.0
	github.com/nats-io/jwt/v2 v2.8.0
	github.com/nats-io/nats.go v1.47.0
	github.com/nats-io/nkeys v0.4.12
	github.com/urfave/cli/v3 v3.8.0
)

replace github.com/aaiken/nats-score/checks => ../checks

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/miekg/dns v1.1.69 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/prometheus-community/pro-bing v0.7.0 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/mod v0.30.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/tools v0.39.0 // indirect
)

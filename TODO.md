# TODO
- [ ] HTTPS
- [ ] Dynamic Timeout per check
- [ ] Setup command to generate server config file
- [ ] Check commands take CLI arguments

## Agent
- If nats restarts and bucket missing spams attempts
  - time=2026-05-20T23:00:45.623-04:00 level=INFO source=config.go:100 msg="Initial KV sync complete, watching for updates..."

## Frontend
- [ ] Grouping NATS data points. Must be a better way

## Server
- [ ] User custom setting changes triggers server checks to reload
- Message when 0 checks returned
- Handlers and routes combined

# TODO
- [ ] HTTPS
- [ ] Dynamic Timeout per check
- [ ] Standardize logging
- [ ] Setup command to generate server config file

## Agent
- Once custom settings get removed checks don't revert to default values
  - Might be desired effect
- If nats restarts and bucket missing spams attempts
  - time=2026-05-20T23:00:45.623-04:00 level=INFO source=config.go:100 msg="Initial KV sync complete, watching for updates..."

## Frontend
- [ ] Admin dashboard
- [ ] Admin settings
- [ ] Observer dashboard
- [ ] Grouping NATS data points. Must be a better way

## Server
- [ ] Additional Admin routes
  - Update settings?
- [ ] Flip static_auth to be token:role (would allow multiple of the same account with different tokens)
- [ ] User custom setting changes triggers server checks to reload
- Message when 0 checks returned
- Handlers and routes combined

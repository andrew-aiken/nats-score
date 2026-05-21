# TODO
- [ ] HTTPS
- [ ] Dynamic Timeout per check

## Agent
- Once custom settings get removed checks don't revert to default values
  - Might be desired effect

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

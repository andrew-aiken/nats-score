# TODO
- [ ] HTTPS
- [ ] Dynamic Timeout per check?
- [x] Checks are not a central file, individual kv per check
- [x] Results have pass/fail in stream name


## Agent
- [x] Dynamic Team resources (ip, team#, etc.)
  - [x] User input can contain templated values. This should be blocked (the janky way)
- [x] Overhaul input instead of env vars (nats, team#)
- [ ] Cleanup main cmd function

## Frontend
- [ ] Admin dashboard
- [ ] Admin settings
  - [x] Start/Stop scoring buttons
  - [x] View global settings
- [ ] Observer dashboard
- [ ] Grouping NATS data points. Must be a better way
- [x] Team ID from JWT

## Server
- [ ] Additional Admin routes
  - Update settings?
- [x] Get team ID during auth
- [ ] Flip static_auth to be token:role (would allow multiple of the same account with different tokens)

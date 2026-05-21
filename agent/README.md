# agent

A distributed scoring agent that executes checks against one or more teams and publishes results to NATS JetStream.

## Usage

```
agent --nats-creds <path> [--team <n> | --teams <expr>] [options]
```

## Flags

| Flag | Alias | Env var | Required | Default | Description |
|---|---|---|---|---|---|
| `--nats-creds` | | `SCORE_NATS_CREDS_FILE` | Yes | | Path to NATS credentials file |
| `--team` | `-t` | `SCORE_TEAM_NUMBER` | One of | | Single team number |
| `--teams` | `-T` | `SCORE_TEAM_NUMBERS` | One of | | Team numbers or ranges (see below) |
| `--nats-address` | | `SCORE_NATS_URL` | No | `nats://127.0.0.1:4222` | NATS server address |
| `--log-level` | `-l` | | No | `info` | Log level: `debug`, `info`, `warn`, `error` |

`--team` and `--teams` are mutually exclusive. Exactly one is required.

### `--teams` syntax

Accepts a comma-separated list of individual numbers and/or inclusive ranges:

| Expression | Teams scored |
|---|---|
| `5` | 5 |
| `1,3,7` | 1, 3, 7 |
| `1-5` | 1, 2, 3, 4, 5 |
| `1,3,5-8,10` | 1, 3, 5, 6, 7, 8, 10 |

Duplicate values are silently deduplicated.

## Examples

```sh
# Single team
agent --team 5 --nats-creds ./creds.nats

# Multiple teams
agent --teams "1,3,7" --nats-creds ./creds.nats

# Range of teams
agent --teams "1-15" --nats-creds ./creds.nats

# Mixed range and individual values
agent --teams "1,3,5-8,10" --nats-creds ./creds.nats

# Custom NATS server
agent --team 1 --nats-creds ./creds.nats --nats-address nats://10.0.0.1:4222

# Via environment variables
SCORE_TEAM_NUMBERS="1-10" SCORE_NATS_CREDS_FILE=./creds.nats agent
```

## How it works

On startup the agent:

1. Connects to the NATS server using JWT + NKey credentials.
2. Opens the `settings` JetStream key-value bucket and watches:
   - `check.*` — global check definitions shared across all teams.
   - `<team>.settings` — per-team attribute overrides, one key per team.
3. Subscribes to `events.score.>` for score trigger events.

When a score event arrives on `events.score.<check-name>`, the agent runs that check for every configured team concurrently and publishes results to:

```
results.<team>.<check-name>.<0|1>
```

where the final segment is `1` for a passing check and `0` for a failing one.

Check definitions and team settings are updated in real time as the key-value store changes — no restart required.

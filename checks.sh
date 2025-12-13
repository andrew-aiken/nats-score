#!/bin/bash

FLAT_JSON=$(jq -c . checks.json)

nats --creds /Users/aaiken/.local/share/nats/nsc/keys/creds/score/score/admin.creds kv put settings settings "$FLAT_JSON"

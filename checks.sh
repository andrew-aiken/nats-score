#!/bin/bash

FLAT_JSON=$(jq -c . checks.json)

nats --user admin --password adminpass kv put settings settings "$FLAT_JSON"

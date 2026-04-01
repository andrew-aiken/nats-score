#!/bin/sh
set -e

NATS_URL="${NATS_URL:-nats://nats:4222}"
NATS_CREDS="${NATS_CREDS:-/etc/nats/admin.creds}"

echo "Waiting for NATS server..."
until nats --creds="$NATS_CREDS" --server="$NATS_URL" account info > /dev/null 2>&1; do
  echo "NATS not ready, retrying in 1s..."
  sleep 1
done
echo "NATS server is ready"

echo "Creating KV bucket 'settings'..."
nats kv add settings \
  --creds="$NATS_CREDS" --server="$NATS_URL" \
  --description="Score data storage" \
  --history=5 \
  --ttl=0 \
  --max-value-size=-1 \
  --max-bucket-size=-1 \
  2>/dev/null || echo "KV bucket 'settings' already exists"

echo "Creating stream 'results'..."
nats stream add results \
  --creds="$NATS_CREDS" --server="$NATS_URL" \
  --subjects="results.>" \
  --description="Stream of score update events" \
  --retention=limits \
  --max-age=30d \
  --storage=file \
  --replicas=1 \
  --discard=old \
  --max-msgs=-1 \
  --max-bytes=-1 \
  --max-msg-size=-1 \
  --dupe-window=2m \
  --no-allow-rollup \
  --deny-delete \
  --deny-purge \
  --defaults \
  2>/dev/null || echo "Stream 'results' already exists"

echo "NATS JetStream setup complete!"
nats kv ls --creds="$NATS_CREDS" --server="$NATS_URL"
nats stream ls --creds="$NATS_CREDS" --server="$NATS_URL" --all

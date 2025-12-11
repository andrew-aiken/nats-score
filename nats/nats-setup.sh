#!/bin/sh
set -e

NATS_URL="${NATS_URL:-nats://nats:4222}"
NATS_USER="${NATS_USER:-admin}"
NATS_PASSWORD="${NATS_PASSWORD:-adminpass}"

# Build credentials flag
CREDS="--user=${NATS_USER} --password=${NATS_PASSWORD}"

echo "Waiting for NATS server..."
until nats --server="$NATS_URL" $CREDS account info > /dev/null 2>&1; do
  echo "NATS not ready, retrying in 1s..."
  sleep 1
done
echo "NATS server is ready"

echo "Creating KV bucket 'settings'..."
nats kv add settings \
  --server="$NATS_URL" $CREDS \
  --description="Score data storage" \
  --history=5 \
  --ttl=0 \
  --max-value-size=-1 \
  --max-bucket-size=-1 \
  2>/dev/null || echo "KV bucket 'settings' already exists"

echo "Creating stream 'results'..."
nats stream add results \
  --server="$NATS_URL" $CREDS \
  --subjects="results.>" \
  --description="Stream of score update events" \
  --retention=limits \
  --max-age=24h \
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
nats kv ls --server="$NATS_URL" $CREDS
nats stream ls --server="$NATS_URL" $CREDS --all

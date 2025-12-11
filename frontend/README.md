# NATS Dashboard

## Setup

```bash
npm install
npm run dev
```

## Configuration

The NATS connection is configured in `src/services/nats.ts`:

- **WebSocket URL**: `ws://localhost:8080`
- **Username**: `user`
- **Password**: `password`

## NATS Server Setup

Ensure your NATS server is running with WebSocket support on port 8080. Example configuration:

```conf
websocket {
  port: 8080
  no_tls: true
}

authorization {
  users = [
    { user: "user", password: "password" }
  ]
}
```

## Features

- Real-time message streaming via WebSocket
- Subscribe to multiple NATS subjects (supports wildcards like `events.>`)
- Connection status indicator
- JSON payload pretty-printing
- Message history (last 100 messages)

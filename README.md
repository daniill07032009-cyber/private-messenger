# Private Messenger

An independent messaging platform for web and Android with a Telegram-like user experience. Not based on Skyline and not an official Telegram client.

## Stack
- Backend: Go, PostgreSQL, WebSocket
- Web: React, TypeScript, Vite
- Android: Kotlin (planned native client)
- Local development: Docker Compose
- CI: GitHub Actions

## Current status
Bootstrap scaffold. Authentication, persistent messaging, groups, media, notifications, and calls are not production-ready implementations yet.

## Run locally
Requirements: Docker Compose, Go 1.23+, Node.js 22+.

```sh
docker compose up --build
```

Web UI: http://localhost:5173
API health: http://localhost:8080/healthz

Never commit secrets. End-to-end encryption requires a reviewed protocol and security audit.

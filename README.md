# Private Messenger

An independent messaging platform for web and Android with a familiar chat-first experience. It is not based on Skyline and is not an official Telegram client.

## Stack
- Backend: Go, PostgreSQL, cookie-based sessions
- Web: React, TypeScript, Vite
- Android: native Kotlin, Jetpack Compose, OkHttp
- Local development: Docker Compose
- CI: GitHub Actions (backend tests, web production build, Android debug APK build)

## Implemented
- Account registration and login; bcrypt password hashes and expiring server-side sessions.
- PostgreSQL migrations applied automatically by the API at startup.
- Authenticated conversation lists and persisted messages with membership checks.
- User search and creation/reuse of direct conversations.
- Group and channel creation.
- Web client uses the real API; active histories refresh periodically to show messages from other sessions.
- Android client uses the same API for authentication, chat lists, message history, sending messages, direct-chat discovery, and group/channel creation.
- CI builds the Android debug APK and uploads it as a workflow artifact when the build succeeds.

## Current limitations
- Web uses authenticated WebSocket delivery with reconnect and a periodic REST fallback. Android also receives WebSocket events and refreshes history every 15 seconds as a fallback; automatic WebSocket reconnect on Android is still pending.
- Group membership invitations, admin tools, public links, attachments, push notifications, voice/video calls, and end-to-end encryption are not implemented.
- Android currently uses an in-memory cookie jar; a process restart may require signing in again.
- The default Android API URL is http://10.0.2.2:8080/, intended for the Android emulator talking to a local development server. For a deployed backend, pass -PMESSENGER_API_URL=https://your-api.example/ to Gradle.
- Production deployment requires HTTPS, managed secrets, backups, monitoring, abuse controls and an independent security review.

## Run locally
Requirements: Docker Compose, Go 1.23+, Node.js 22+. To start the web client and API with PostgreSQL:

```sh
docker compose up --build
```

Web UI: http://localhost:5173  
API health: http://localhost:8080/healthz

The debug APK can be obtained from a successful Android APK build workflow run under Actions → Artifacts. It is not a signed Play Store release. Never commit secrets.

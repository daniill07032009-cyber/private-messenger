# Architecture overview

## Current implementation
- Go HTTP API exposes `GET /healthz` and `GET /api/v1`.
- React web UI is a local-only interactive mock. It does not authenticate users or deliver messages to another device.
- Android app is a native Kotlin/Jetpack Compose starter screen with demo chats.
- SQL migration defines the first relational model, but migrations and PostgreSQL access are not yet wired into the API.
- GitHub Actions runs Go tests/vet and the web TypeScript production build.

## Planned service boundaries
1. Identity and session management
2. Conversations, membership and authorization
3. Message persistence and real-time delivery
4. Media upload/download authorization
5. Notifications and device registrations
6. Calls/signaling integration

## Non-negotiable security properties
- Authorization checks happen server-side for every read and write.
- Passwords are stored only as hashes using a maintained password-hashing library.
- Uploaded media uses scoped authorization and size/type limits.
- Rate limiting and abuse reporting are designed before public launch.
- End-to-end encryption is a separate security project requiring a reviewed protocol and audit.

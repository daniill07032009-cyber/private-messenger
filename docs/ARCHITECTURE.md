# Architecture overview

## Implemented in this stage
- Go API backed by PostgreSQL.
- Startup SQL migrations embedded in the API binary.
- Registration/login with bcrypt password hashes.
- Random session tokens stored only as SHA-256 hashes in PostgreSQL; HttpOnly, SameSite=Lax cookies expire after 30 days.
- Authenticated conversation creation and membership checks for message reads/writes.
- React API client helper for the REST contract.
- PostgreSQL included in Docker Compose.

## Current limitations
- The web UI still renders demo data and is not yet wired to the authenticated API.
- Direct-message creation, user discovery and account recovery are not implemented.
- No WebSocket delivery, media, push notifications, voice/video calls or end-to-end encryption yet.
- Production requires HTTPS, managed secrets, backups, monitoring and abuse controls.

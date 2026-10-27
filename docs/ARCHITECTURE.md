# Architecture overview

## Implemented
- Go API backed by PostgreSQL; migrations are embedded in the API and run at startup.
- Registration/login with bcrypt password hashes.
- Random session tokens stored only as SHA-256 hashes in PostgreSQL; HttpOnly, SameSite=Lax cookies expire after 30 days.
- Authenticated user discovery and direct-conversation creation/reuse.
- Group/channel creation and membership-protected message reads/writes.
- React client connected to the REST API for auth, conversations, and message history/send.
- Android Kotlin/Compose client connected to the same API for auth, chat lists, message history/send, direct chat discovery, and group/channel creation.
- CI covers Go tests/vet, web production build, and Android debug APK assembly/artifact upload.

## Current limitations
- Cross-session message visibility uses periodic refresh, not WebSocket delivery.
- Direct chats do not yet have full member-management UI; group/channel invitation, roles management, ownership transfer, and public/private links are pending.
- Media uploads, notifications, voice/video calls, and end-to-end encryption are not implemented.
- Android session cookies are held in memory and may be lost when the app process ends.
- Production requires HTTPS, managed secrets, backups, monitoring, rate limiting, abuse controls, and independent security review.

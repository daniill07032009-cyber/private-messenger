# Roadmap

This is a staged plan, not a claim that all Telegram features already exist.

## Phase 1 — foundations
- [x] Repository scaffold and CI
- [x] Minimal Go health endpoint
- [x] Initial web chat-list and conversation UI (demo data only)
- [ ] Configuration validation and structured logging
- [ ] PostgreSQL migrations and repository layer

## Phase 2 — accounts and direct messages
- [ ] Registration, login, verification strategy
- [ ] Password hashing, session rotation, logout and device sessions
- [ ] Profiles, usernames, contacts and blocking
- [ ] Persisted conversations and messages
- [ ] Authenticated WebSocket, delivery acknowledgements and reconnect

## Phase 3 — messaging
- [ ] Replies, forwards, edits, deletes and read states
- [ ] Reactions, pinned messages, drafts and search
- [ ] Attachments, upload authorization and limits
- [ ] Message pagination and abuse controls

## Phase 4 — groups and channels
- [ ] Owners, administrators, roles and permissions
- [ ] Public/private visibility and invite links
- [ ] Moderation, reports, mute and member management
- [ ] Channel publishing and subscriber delivery

## Phase 5 — Android
- [ ] Native Kotlin app and secure token storage
- [ ] Device registration, push notifications and deep links
- [ ] Shared API contract and integration tests

## Phase 6 — calls and production
- [ ] Voice messages and cancellation before send
- [ ] WebRTC audio/video calls and TURN
- [ ] Push delivery, observability, backups and recovery
- [ ] Threat modeling, independent security review and load tests

## Security requirements
- Use reviewed password hashing and secure session handling.
- Enforce server-side authorization for every conversation and attachment.
- Do not invent custom end-to-end encryption.
- Never commit production credentials.

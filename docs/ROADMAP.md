# Roadmap

## Foundations and account API
- [x] Go API and health endpoint
- [x] PostgreSQL schema and startup migrations
- [x] Registration/login with bcrypt
- [x] Hashed expiring sessions in HttpOnly cookies
- [x] Authenticated group/channel creation and membership-protected message endpoints
- [x] Integration tests against PostgreSQL for registration and message delivery
- [x] User search and direct chat creation/reuse
- [ ] Rate limiting, email/phone verification and account recovery

## Messaging clients
- [x] Connect web UI to authenticated API
- [x] Direct messages and user discovery
- [x] Native Android client connected to shared API
- [x] CI build for Android debug APK
- [x] Android WebSocket message subscription with REST fallback
- [x] Automatic Android WebSocket reconnect with a 2-second retry delay
- [x] Authenticated WebSocket delivery for the web client with reconnect and REST fallback
- [ ] Delivery acknowledgements and read receipts
- [ ] Read receipts, edits, deletes, replies, forwards and reactions
- [x] Basic in-chat message search in web and Android clients
- [ ] Pagination, drafts, pins and attachment uploads

## Groups and channels
- [x] Create groups and channels
- [ ] Owner/admin permissions and ownership transfer
- [ ] Public/private visibility and invite links
- [ ] Moderation, reporting, muting and member management
- [ ] Channel publishing and subscriber delivery

## Android
- [x] Login, registration, conversation list and persisted messages
- [x] Direct chat discovery and group/channel creation
- [ ] Secure persistent device sessions, push notifications and offline sync
- [ ] Release signing and device/emulator smoke tests

## Production
- [ ] Voice messages and WebRTC calls/TURN
- [ ] Observability, backups, abuse prevention and recovery drills
- [ ] Threat modeling and independent security review
- [ ] Reviewed end-to-end encryption protocol

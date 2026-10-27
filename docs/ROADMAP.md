# Roadmap

## Foundations and account API
- [x] Go API and health endpoint
- [x] PostgreSQL schema and startup migrations
- [x] Registration and login with bcrypt
- [x] Hashed expiring sessions in HttpOnly cookies
- [x] Authenticated group/channel creation and membership-protected message endpoints
- [x] Web API client contract
- [ ] Integration tests against PostgreSQL
- [ ] Rate limiting, verification and account recovery

## Real-time messaging
- [ ] Connect web UI to authenticated API
- [ ] Direct messages and user discovery
- [ ] WebSocket delivery, acknowledgements and reconnect
- [ ] Read receipts, edits, deletes, replies, forwards and reactions
- [ ] Pagination, drafts, pins, search and attachment uploads

## Groups and channels
- [ ] Owner/admin permissions and ownership transfer
- [ ] Public/private visibility and invite links
- [ ] Moderation, reporting, muting and member management
- [ ] Channel publishing and subscriber delivery

## Android
- [ ] Connect native Kotlin app to the shared API
- [ ] Device sessions, push notifications and offline sync

## Production
- [ ] Voice messages and WebRTC calls/TURN
- [ ] Observability, backups, abuse prevention and recovery drills
- [ ] Threat modeling and independent security review

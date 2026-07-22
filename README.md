# Underground Task Marketplace

[![Backend CI](https://github.com/voduybaokhanh/task_under/actions/workflows/backend-ci.yml/badge.svg)](https://github.com/voduybaokhanh/task_under/actions/workflows/backend-ci.yml)
[![Mobile CI](https://github.com/voduybaokhanh/task_under/actions/workflows/mobile-ci.yml/badge.svg)](https://github.com/voduybaokhanh/task_under/actions/workflows/mobile-ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A privacy-focused, anonymous task-for-reward marketplace built with Go (backend) and React Native/Expo (mobile).

## Overview

This is an anonymous task marketplace where:

- Users remain anonymous (device-based identity, no email/password)
- Task owners post tasks with monetary rewards
- Claimers can claim and complete tasks
- Escrow system handles payments
- Real-time chat for task communication
- Reputation system that preserves anonymity

## Architecture

### Backend (Go)

**Layers:**

- **Domain**: Core business entities (User, Task, Claim, Chat, Escrow)
- **Repository**: Data access layer (PostgreSQL)
- **Service**: Business logic layer
- **Handler**: HTTP request handlers (Gin)
- **WebSocket**: Real-time communication hub (Redis Pub/Sub fanout across instances)

**Key Design Decisions:**

- Clean architecture with clear separation of concerns
- Repository pattern for testability
- Service layer encapsulates business rules
- WebSocket hub for real-time updates
- Background job for auto-cancelling expired tasks
- Chat messages are stored as ciphertext: the server routes them but cannot read them

### Mobile (React Native + Expo)

**Structure:**

- **Screens**: UI components (TaskList, TaskDetail, CreateTask, Chat)
- **Store**: Zustand state management
- **Services**: API client and WebSocket client
- **Types**: TypeScript type definitions

**Key Design Decisions:**

- Zustand for simple, performant state management
- Device-based authentication (X-Device-ID header)
- AsyncStorage for device ID persistence
- Tab navigation for main flows

## Database Schema

### Core Tables

- **users**: Anonymous users (device_id based)
- **tasks**: Task listings with deadlines and rewards
- **claims**: User claims on tasks
- **chats**: Anonymous chat threads
- **messages**: Chat messages
- **escrow_transactions**: Payment tracking
- **arbitrations**: Dispute resolution (extensible)

### Key Constraints

- Task claim deadlines must be before owner deadlines
- Claim limits enforced at database level
- Escrow locked on task creation
- Chat deletion is soft (flags for both participants)

## Core Business Rules

1. **Anonymity**: No email/password, device-based identity only
2. **Task Lifecycle**:
   - Created with escrow locked
   - Auto-cancels if no claims by claim deadline
   - Owner approves/rejects completion
3. **Claiming**:
   - Enforced server-side limits
   - First claim updates task status to "claimed"
4. **Escrow**:
   - Locked on creation
   - Released on approval
   - Refunded on cancellation
5. **Chat**:
   - Opens on completion submission
   - Deletion removes for both participants
   - Re-opening creates new thread
   - End-to-end encrypted (NaCl box / X25519); the backend stores only ciphertext

## Setup & Running

### Prerequisites

- Go 1.22+
- Node.js 20+ (22.6+ to run `npm test`, which uses Node's native TypeScript support)
- Docker & Docker Compose
- PostgreSQL 15–18 (or use Docker)
- Redis (or use Docker)

### Backend Setup

1. **Start dependencies:**

```bash
docker-compose up -d postgres redis
```

2. **Run migrations:**

```bash
# Install migrate tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path backend/migrations -database "postgres://postgres:postgres@localhost:5432/task_underground?sslmode=disable" up
```

3. **Set environment variables:**

```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/task_underground?sslmode=disable"
export PORT=8080
```

4. **Run backend:**

```bash
cd backend
go mod download
go run cmd/server/main.go
```

Backend will be available at `http://localhost:8080`

### Mobile Setup

1. **Install dependencies:**

```bash
cd mobile
npm install
```

2. **Configure API URL:**
   Create `.env` file:

```
EXPO_PUBLIC_API_URL=http://localhost:8080
```

3. **Run mobile app:**

```bash
npm start
```

Then press `i` for iOS simulator or `a` for Android emulator.

### Docker Compose (Full Stack)

```bash
docker compose up
```

This starts:

- PostgreSQL on port 5432 (set `POSTGRES_PORT` if that port is already taken)
- Redis on port 6379
- Backend on port 8080
- Prometheus on port 9090
- Grafana on port 3000 (admin/admin)

Running several backend instances against the same Redis is supported: WebSocket
events are fanned out over Pub/Sub, so a user connected to one instance still
receives events emitted by another.

## API Endpoints

### Users

- `GET /api/v1/users/me` - Get current user profile (reputation, earnings, spending)
- `PUT /api/v1/users/me/push-token` - Register the device's Expo push token
- `PUT /api/v1/users/me/pubkey` - Publish this device's X25519 public key (E2EE)
- `GET /api/v1/users/:id/pubkey` - Fetch another user's public key

### Tasks

- `POST /api/v1/tasks` - Create task
- `GET /api/v1/tasks` - List open tasks
- `GET /api/v1/tasks/my` - Get user's tasks
- `GET /api/v1/tasks/search?q=&status=` - Full-text search (PostgreSQL tsvector)
- `GET /api/v1/task/:id` - Get task details

### Claims

- `POST /api/v1/tasks/:task_id/claims` - Claim a task
- `GET /api/v1/tasks/:task_id/claims` - Get claims for task
- `GET /api/v1/claims/:id` - Get claim details
- `POST /api/v1/claims/:id/submit` - Submit completion
- `POST /api/v1/claims/:id/approve` - Approve claim (owner)
- `POST /api/v1/claims/:id/reject` - Reject claim (owner)

### Chat

- `GET /api/v1/tasks/:task_id/chats` - Get chats for task
- `POST /api/v1/tasks/:task_id/chats` - Get or create chat
- `DELETE /api/v1/chats/:id` - Delete chat
- `POST /api/v1/chats/:id/messages` - Send message
- `GET /api/v1/chats/:id/messages` - Get messages

### WebSocket

- `GET /ws` - WebSocket connection (requires X-Device-ID header)

Events pushed to the client: `chat_message`, `claim_created`, `completion_submitted`,
`claim_approved`, `claim_rejected`. The same events are sent as Expo push
notifications when the user has registered a push token.

### Uploads

- `POST /api/v1/upload/presign` - Returns a presigned S3 PUT URL (15 min) plus the eventual public URL. Images only (jpeg/png/webp); returns 503 when `AWS_BUCKET_NAME` is unset

### Operations

- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics (requests, latency, active WebSockets, task counters)

## Testing

Run backend tests (no external services needed — Redis is faked in-process
with miniredis):

```bash
cd backend
go test ./... -race
```

Mobile type check and unit tests (`node --test`, no test framework):

```bash
cd mobile
npx tsc --noEmit
npm test
```

With a backend running on :8080, the encrypted-chat path can be checked
end to end:

```bash
npm run test:e2e
```

Key test coverage:

- Task auto-cancellation on expired deadlines
- Claim limit enforcement
- Escrow locking/releasing
- WebSocket fanout across two instances over Redis, with no duplicate on the publisher
- Notifications: who gets told about a chat message or claim, and what reaches Expo
- E2EE: round trip, tamper detection, wrong-key rejection, and that the server
  only ever stores ciphertext
- Presigned uploads against a real S3 API (MinIO): upload succeeds, the object
  is publicly readable, keys never collide, and expired URLs are refused

## Production Considerations

### Security

- [x] Per-device rate limiting (Redis sliding window, 60 req/min)
- [ ] Implement proper CORS configuration
- [ ] Add request validation middleware
- [ ] Secure WebSocket connections (WSS)
- [ ] Add input sanitization
- [x] Image upload with content-type validation (presigned S3 URLs)

### Scalability

- [ ] Add database connection pooling
- [ ] Implement Redis caching for frequently accessed data
- [ ] Add message queue for background jobs
- [x] Horizontal scaling for WebSocket connections (Redis Pub/Sub fanout)
- [ ] Database read replicas

### Monitoring

- [ ] Add structured logging
- [x] Metrics collection (Prometheus + Grafana dashboards)
- [ ] Error tracking (Sentry)
- [x] Health check endpoints

### Notifications

- [x] Push notifications via Expo (claim, approval, rejection, submission, chat)
- [ ] Notification preferences per user

### Payment Integration

- [ ] Integrate real payment processor (Stripe, etc.)
- [ ] Implement actual escrow service
- [ ] Add payment webhooks

### Image Storage

- [x] Implement image upload to S3/Cloud Storage (presigned PUT, client uploads direct)
- [ ] Add image validation and processing (size limits, re-encoding)
- [ ] CDN for image delivery

## Changelog

### v3 — Production Upgrades

**CI/CD**
- GitHub Actions: `backend-ci` (vet, lint, test, docker build), `mobile-ci` (npm ci, tsc, expo export), `release` (tag `v*` → image on GHCR)

**Rate limiting**
- Per-device Redis sliding window (60 req/min) replaces the global limiter; returns `429` with `X-RateLimit-Remaining` and `Retry-After`
- No-op when Redis is absent, so local development needs no extra service

**Observability**
- Prometheus metrics: request counter, latency histogram, active WebSocket gauge, tasks created/completed
- `/metrics` endpoint plus Prometheus and Grafana services with provisioned dashboards

**Search**
- `tasks.search_vector` (tsvector + GIN index, kept fresh by a trigger) and `GET /api/v1/tasks/search`, ranked with `ts_rank`
- Mobile search bar calls the API with a 300 ms debounce instead of filtering locally

**WebSocket scaling**
- Hub fans out over Redis Pub/Sub (`ws:fanout`), so any instance can deliver to a user connected to any other; falls back to in-memory when Redis is absent
- Fixed a latent data race: the broadcast paths mutated the client map under a read lock
- The hub previously had no callers at all — services now emit events through a `Notifier` interface

**Image upload**
- `POST /api/v1/upload/presign` hands out a 15-minute presigned S3 PUT URL; the file goes straight from the device to the bucket, so AWS credentials never leave the backend and no image bytes pass through it
- Object keys are generated server-side (UUID under `task-images/`), so one client cannot overwrite another's image
- Works with any S3-compatible service via `AWS_ENDPOINT_URL` — the tests run against MinIO, in CI too
- Mobile: pick a photo in Create Task, preview it, and see it on the task detail screen

**End-to-end encrypted chat**
- Each device generates an X25519 key pair on first launch; the secret key stays in the OS keystore (`expo-secure-store`) and is never uploaded
- Public keys are published to `users.public_key`; opening a chat fetches the other party's key and derives a shared secret with `nacl.box.before`
- Messages travel as `E2E1.<nonce>.<ciphertext>` — the backend, the database and the push payload only ever hold ciphertext
- Chat header shows `🔒 E2E Encrypted`, or `🔓 Not encrypted` when the other side has not published a key yet; pre-E2EE plaintext messages still render

**Push notifications**
- Expo push (no Firebase credentials required): `users.push_token`, `PUT /api/v1/users/me/push-token`, and `registerForPushNotifications()` on app start
- Claim, approval, rejection, completion and chat events reach both the open app (WebSocket) and the closed one (push) via `MultiNotifier`
- Push copy is generic ("Bạn có tin nhắn mới"), so an encrypted chat leaks nothing through the notification tray

**Fixes**
- The mobile app never opened its WebSocket connection: `WebSocketService` existed but had no callers. It is now connected at startup and routes `chat_message` into the chat store
- Chat bubbles compared `sender_id` against the *device* ID, so no message was ever recognised as ours and everything rendered left-aligned

### v2 — UI/UX Level Up & PostgreSQL 18 Compatibility

#### Mobile

**New screens & navigation**
- **Profile screen** — anonymous avatar, reputation progress bar, stats grid (Total Earned, Total Spent, Tasks Created, Completed, Open)
- Tab bar now has icons (Ionicons: list, briefcase, person) with green active tint and dark styling

**Task List (Explore tab)**
- Search bar with real-time keyword filtering across title and description
- Filter chips: All / Open / Claimed / Completed
- Task cards show colored status badge (green/orange/blue/red), reward, and time remaining ("9d left", "2h left")

**Create Task**
- Claim Deadline: preset chips — 1 day / 3 days / 7 days / 14 days
- Completion Deadline: preset chips — 7 days / 14 days / 30 days / 60 days
- Max Claimants: quick-select chips (1 / 2 / 3 / 5 / 10)
- Deadline hint shows computed date below each selection

**Chat**
- Sender bubbles aligned right (green), receiver bubbles aligned left (dark gray)
- Disabled send button when input is empty
- Submit on keyboard return key

**My Tasks**
- Header summary: "X open · Y completed"
- Status badges consistent with Explore tab

#### Backend

**New endpoint**
- `GET /api/v1/users/me` — returns current user's profile (id, reputation, total\_earned, total\_spent)

**PostgreSQL 18 compatibility fixes**
- All UUID parameters across every repository query now use explicit `::uuid` cast and `.String()` serialization — required because `lib/pq`'s extended query protocol no longer infers UUID types automatically in PostgreSQL 18
- `UpdateTransactionStatus`: split `$1` reuse in CASE WHEN into a separate `$2` parameter to avoid type-inference conflict
- `CompletionText` / `CompletionImageURL` now scanned via `sql.NullString` to handle NULL values before submission

**Repository refactor**
- Extracted `scanTask`, `scanClaim`, `scanChat`, `scanUser` helpers to eliminate repetitive scan boilerplate

---

## Known Limitations

1. **Escrow**: Currently simulated, not real payment processing
2. **Image Upload**: Requires an S3 bucket; no server-side size limit or re-encoding yet
3. **Arbitration**: Owner-only, no third-party arbitration yet
4. **Image upload**
- `POST /api/v1/upload/presign` hands out a 15-minute presigned S3 PUT URL; the file goes straight from the device to the bucket, so AWS credentials never leave the backend and no image bytes pass through it
- Object keys are generated server-side (UUID under `task-images/`), so one client cannot overwrite another's image
- Works with any S3-compatible service via `AWS_ENDPOINT_URL` — the tests run against MinIO, in CI too
- Mobile: pick a photo in Create Task, preview it, and see it on the task detail screen

**End-to-end encrypted chat**
- Each device generates an X25519 key pair on first launch; the secret key stays in the OS keystore (`expo-secure-store`) and is never uploaded
- Public keys are published to `users.public_key`; opening a chat fetches the other party's key and derives a shared secret with `nacl.box.before`
- Messages travel as `E2E1.<nonce>.<ciphertext>` — the backend, the database and the push payload only ever hold ciphertext
- Chat header shows `🔒 E2E Encrypted`, or `🔓 Not encrypted` when the other side has not published a key yet; pre-E2EE plaintext messages still render

**Push notifications**: Expo push service only; no Firebase/APNs credentials of our own
5. **E2EE key trust**: public keys are served by the backend and taken on
   trust — no out-of-band verification (safety numbers), so a malicious server
   could substitute a key. Losing the device loses the message history, by design

## Future Improvements

1. **Third-party Arbitration**: Extensible arbitration system
2. **Task Categories**: Organize tasks by category
3. **Search & Filters**: Full-text search, filters by reward, deadline
4. **Notifications**: Push notifications for task updates
5. **Reputation System**: More sophisticated reputation algorithm
6. **Task Templates**: Reusable task templates
7. **Bulk Operations**: Batch claim approval/rejection

## License

MIT

# Underground Task Marketplace

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
- **WebSocket**: Real-time communication hub

**Key Design Decisions:**

- Clean architecture with clear separation of concerns
- Repository pattern for testability
- Service layer encapsulates business rules
- WebSocket hub for real-time updates
- Background job for auto-cancelling expired tasks

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

## Setup & Running

### Prerequisites

- Go 1.22+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 15+ (or use Docker)
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
docker-compose up
```

This starts:

- PostgreSQL on port 5432
- Redis on port 6379
- Backend on port 8080

## API Endpoints

### Tasks

- `POST /api/v1/tasks` - Create task
- `GET /api/v1/tasks` - List open tasks
- `GET /api/v1/tasks/my` - Get user's tasks
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

## Testing

Run backend tests:

```bash
cd backend
go test ./internal/service/...
```

Key test coverage:

- Task auto-cancellation on expired deadlines
- Claim limit enforcement
- Escrow locking/releasing

## Production Considerations

### Security

- [ ] Add rate limiting per user (currently global)
- [ ] Implement proper CORS configuration
- [ ] Add request validation middleware
- [ ] Secure WebSocket connections (WSS)
- [ ] Add input sanitization
- [ ] Implement image upload with validation

### Scalability

- [ ] Add database connection pooling
- [ ] Implement Redis caching for frequently accessed data
- [ ] Add message queue for background jobs
- [ ] Horizontal scaling for WebSocket connections
- [ ] Database read replicas

### Monitoring

- [ ] Add structured logging
- [ ] Metrics collection (Prometheus)
- [ ] Error tracking (Sentry)
- [ ] Health check endpoints

### Payment Integration

- [ ] Integrate real payment processor (Stripe, etc.)
- [ ] Implement actual escrow service
- [ ] Add payment webhooks

### Image Storage

- [ ] Implement image upload to S3/Cloud Storage
- [ ] Add image validation and processing
- [ ] CDN for image delivery

## Known Limitations

1. **Escrow**: Currently simulated, not real payment processing
2. **Image Upload**: Placeholder only, needs S3/Cloud Storage integration
3. **Arbitration**: Owner-only, no third-party arbitration yet
4. **Rate Limiting**: Global rate limit, should be per-user
5. **WebSocket**: Single instance only, needs Redis pub/sub for scaling

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

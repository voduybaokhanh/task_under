# Architecture Documentation

## Project Understanding & Assumptions

### Business Goal
An anonymous task-for-reward marketplace where users can post tasks with monetary rewards, and other users can claim and complete them. The system emphasizes privacy and anonymity while maintaining trust through escrow and reputation systems.

### Key Assumptions
1. **Anonymity First**: No email/password authentication - device-based identity only
2. **Escrow Simulation**: Current implementation simulates escrow; real payment integration needed for production
3. **Owner Arbitration**: Initial MVP uses owner-only arbitration; extensible for third-party arbitrators
4. **Single Instance**: WebSocket hub designed for single instance; needs Redis pub/sub for horizontal scaling
5. **Image Storage**: Placeholder for image URLs; requires S3/Cloud Storage integration

## Current State Audit

### What Exists
✅ Complete database schema with migrations
✅ Full backend architecture (domain, repository, service, handler layers)
✅ REST API endpoints for all core features
✅ WebSocket server for real-time updates
✅ Mobile app with React Native + Expo
✅ State management with Zustand
✅ Docker Compose setup
✅ Basic test coverage

### What's Missing/Broken
⚠️ Real payment/escrow integration (simulated)
⚠️ Image upload functionality (placeholder)
⚠️ Production-ready error handling
⚠️ Comprehensive test coverage
⚠️ Rate limiting per user (currently global)
⚠️ WebSocket scaling (single instance only)

### What Must Be Refactored
- None critical for MVP, but consider:
  - Add transaction support for multi-step operations
  - Improve error messages for better UX
  - Add request validation middleware

## Final MVP Scope

### MUST-HAVE (Implemented)
- ✅ Anonymous user creation (device-based)
- ✅ Task creation with escrow locking
- ✅ Task claiming with limit enforcement
- ✅ Completion submission (text + image URL)
- ✅ Owner approval/rejection
- ✅ Chat system (anonymous, deletable)
- ✅ Auto-cancellation of expired tasks
- ✅ Escrow refund on cancellation
- ✅ Reputation tracking
- ✅ Real-time updates via WebSocket

### NICE-TO-HAVE (Future)
- Third-party arbitration
- Task categories and search
- Push notifications
- Image upload with validation
- Advanced reputation algorithm
- Task templates

## Architecture Design

### Backend Architecture

```
┌─────────────────────────────────────────┐
│           HTTP Handlers (Gin)            │
│  - TaskHandler, ClaimHandler, ChatHandler│
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         Service Layer                    │
│  - TaskService, ClaimService, etc.      │
│  - Business logic & validation           │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│      Repository Layer                     │
│  - UserRepo, TaskRepo, ClaimRepo, etc.  │
│  - Database access abstraction           │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         Domain Models                    │
│  - User, Task, Claim, Chat, Escrow      │
└─────────────────────────────────────────┘
```

**Justification:**
- Clean separation allows easy testing and maintenance
- Repository pattern enables database-agnostic business logic
- Service layer encapsulates complex business rules
- Handlers remain thin, focused on HTTP concerns

### Mobile Architecture

```
┌─────────────────────────────────────────┐
│              Screens (UI)                │
│  - TaskList, TaskDetail, CreateTask     │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         State Management (Zustand)      │
│  - useTaskStore, useChatStore            │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         Services Layer                   │
│  - ApiService, WebSocketService          │
└─────────────────────────────────────────┘
```

**Justification:**
- Zustand chosen for simplicity and performance
- Service layer abstracts API calls
- Screens remain presentation-focused
- TypeScript ensures type safety

### Database Design

**Key Tables:**
- `users`: Device-based anonymous identity
- `tasks`: Task listings with deadlines
- `claims`: User claims on tasks
- `chats`: Anonymous chat threads
- `messages`: Chat messages
- `escrow_transactions`: Payment tracking

**Constraints:**
- Foreign keys ensure referential integrity
- Check constraints validate business rules
- Indexes on frequently queried columns
- Soft deletes for chats (deletion flags)

## Code Quality

### Backend
- ✅ Consistent error handling
- ✅ Input validation
- ✅ Transaction support where needed
- ✅ Concurrency-safe claim limits
- ⚠️ Needs more comprehensive tests

### Mobile
- ✅ TypeScript for type safety
- ✅ Clean component structure
- ✅ State management separation
- ⚠️ Needs error boundary handling
- ⚠️ Needs offline support consideration

## Testing Strategy

### Current Coverage
- Task auto-cancellation logic
- Claim limit enforcement
- Basic service layer tests

### Needed
- Integration tests for API endpoints
- Repository layer tests
- WebSocket connection tests
- Mobile component tests

## Deployment Considerations

### Development
- Docker Compose for local setup
- Hot reload for backend (if using air/realize)
- Expo for mobile development

### Production
- Database migrations via CI/CD
- Environment variable management
- Health check endpoints
- Graceful shutdown handling
- Logging and monitoring

## Security Considerations

### Current
- Device-based authentication
- Input validation
- SQL injection prevention (parameterized queries)
- CORS configuration

### Needed
- Rate limiting per user
- Request size limits
- Image upload validation
- XSS prevention
- CSRF protection (if adding web interface)

## Performance Considerations

### Current
- Database indexes on key columns
- Connection pooling ready
- Efficient queries

### Needed
- Redis caching layer
- Database read replicas
- WebSocket connection pooling
- Image CDN

## Remaining Risks

1. **Escrow**: Not real payment processing - critical for production
2. **Scalability**: WebSocket hub doesn't scale horizontally
3. **Image Storage**: No actual upload/storage implementation
4. **Rate Limiting**: Global limit, not per-user
5. **Error Handling**: Could be more user-friendly

## Future Improvements

1. **Payment Integration**: Real escrow service (Stripe, etc.)
2. **Image Upload**: S3/Cloud Storage with validation
3. **Arbitration**: Third-party arbitrator system
4. **Search**: Full-text search on tasks
5. **Notifications**: Push notifications for updates
6. **Analytics**: Task completion rates, user behavior
7. **Mobile**: Offline support, better error handling

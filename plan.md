# Task Underground — Implementation Plan

Mục tiêu: nâng cấp project lên production-grade để có thêm điểm mạnh trên CV.  
Thứ tự ưu tiên: impact cao → thấp, phụ thuộc kỹ thuật làm trước.

## Tiến độ

- [x] GitHub Actions CI/CD — commit `461f16f`
- [x] Per-user rate limiting (Redis sliding window) — commit `8276abd`
- [x] Prometheus + Grafana — commit `e190ac1`
- [x] Full-text search (tsvector) — commit `9f4e9cd`
- [x] Redis Pub/Sub WebSocket scaling — commit `552be30`, verify 2 instance thật + negative control không Redis
- [x] Push notifications (Expo, không cần credentials Firebase) — kèm việc wire event vào hub (trước đó hub không có call site nào)
- [x] E2EE chat — X25519 + nacl.box, khoá bí mật ở SecureStore, server chỉ giữ ciphertext
- [ ] Image upload S3 — cần AWS credentials
- [ ] Stripe payment — cần Stripe keys

---

## Phase 1 — Redis Pub/Sub cho WebSocket Scaling

**Tại sao ấn tượng:** thể hiện distributed systems — nhiều backend instance vẫn broadcast đúng user.  
**Files liên quan:** `backend/internal/websocket/hub.go`, `backend/cmd/server/main.go`

### Bước thực hiện

1. **Thêm Redis client vào `main.go`**
   - Parse `REDIS_URL` env (đã có trong docker-compose)
   - Dùng `github.com/redis/go-redis/v9`

2. **Tạo `backend/internal/websocket/redis_hub.go`**
   - Wrap Hub hiện tại, thêm 2 Redis channel: `ws:broadcast` và `ws:user:{userID}`
   - `BroadcastToUser`: publish lên Redis thay vì gửi trực tiếp
   - `BroadcastToTask`: publish với danh sách userID vào `ws:task:{taskID}`
   - Goroutine subscriber nhận từ Redis, tìm client local và gửi

3. **Cập nhật `Hub.Run()`**
   - Subscribe Redis channels khi Hub start
   - Giữ nguyên logic in-memory cho single-instance (fallback)

4. **Cập nhật `main.go`**
   - Inject Redis client vào Hub
   - Thêm env flag `REDIS_URL` (optional, nếu không có thì dùng in-memory như cũ)

5. **Test**
   - Chạy 2 instance backend trên port 8080 và 8081
   - Connect WebSocket vào instance 1, send message từ instance 2 → phải nhận được

---

## Phase 2 — Per-User Rate Limiting với Redis Sliding Window

**Tại sao ấn tượng:** thay global rate limiter bằng per-device sliding window — thực tế hơn nhiều.  
**Files liên quan:** `backend/cmd/server/main.go` (phần rate limiter hiện tại)

### Bước thực hiện

1. **Tạo `backend/internal/middleware/rate_limit.go`**
   - Dùng Redis `ZADD` + `ZREMRANGEBYSCORE` + `ZCARD` — sliding window 1 phút
   - Key: `rate:{deviceID}`, window: 60s, limit: 60 requests/phút
   - Return 429 với header `X-RateLimit-Remaining` và `Retry-After`

2. **Xóa global `rate.Limiter` trong `main.go`**
   - Thay bằng `middleware.PerUserRateLimit(redisClient, 60, time.Minute)`
   - Apply sau `AuthMiddleware` (đã có device ID)

3. **Test**
   - Script gửi 70 request liên tiếp → 60 thành công, 10 bị 429
   - Hai device khác nhau không ảnh hưởng nhau

---

## Phase 3 — Full-Text Search với PostgreSQL tsvector

**Tại sao ấn tượng:** thể hiện SQL expertise, không cần infra thêm (dùng PostgreSQL sẵn có).  
**Files liên quan:** `backend/migrations/`, `backend/internal/repository/task_repository.go`, `mobile/app/screens/TaskListScreen.tsx`

### Bước thực hiện

1. **Migration mới: `000002_add_search_vector.up.sql`**
   ```sql
   ALTER TABLE tasks ADD COLUMN search_vector tsvector;
   CREATE INDEX tasks_search_idx ON tasks USING GIN(search_vector);
   UPDATE tasks SET search_vector = to_tsvector('english', title || ' ' || description);
   CREATE TRIGGER tasks_search_update BEFORE INSERT OR UPDATE ON tasks
     FOR EACH ROW EXECUTE FUNCTION
     tsvector_update_trigger(search_vector, 'pg_catalog.english', title, description);
   ```

2. **Thêm `SearchTasks(ctx, query string, status TaskStatus) ([]Task, error)` vào `TaskRepository`**
   - SQL: `WHERE search_vector @@ plainto_tsquery('english', $1)`
   - Kết hợp filter status nếu có
   - Order by `ts_rank(search_vector, ...)` DESC

3. **Thêm `SearchTasks` vào `TaskService`**

4. **Thêm endpoint `GET /api/v1/tasks/search?q=&status=`**
   - Handler trong `task_handler.go`

5. **Cập nhật `TaskListScreen.tsx`**
   - Search bar gọi endpoint search thay vì filter local
   - Debounce 300ms trước khi gọi API

---

## Phase 4 — Observability: Prometheus + Grafana

**Tại sao ấn tượng:** production-readiness — phần lớn side project không có monitoring.  
**Files liên quan:** `backend/cmd/server/main.go`, `docker-compose.yml`

### Bước thực hiện

1. **Thêm `github.com/prometheus/client_golang` vào `go.mod`**

2. **Tạo `backend/internal/metrics/metrics.go`**
   - `http_requests_total` — counter theo method, path, status code
   - `http_request_duration_seconds` — histogram
   - `ws_connections_active` — gauge số WebSocket hiện tại
   - `tasks_created_total`, `tasks_completed_total` — business metrics

3. **Tạo Prometheus middleware trong `backend/internal/middleware/metrics.go`**
   - Wrap mọi request, ghi counter + histogram

4. **Thêm `/metrics` endpoint vào `main.go`**
   - `r.GET("/metrics", gin.WrapH(promhttp.Handler()))`

5. **Cập nhật `docker-compose.yml`**
   - Thêm service `prometheus` (scrape `/metrics` mỗi 15s)
   - Thêm service `grafana` port 3000
   - Volume cho Prometheus config + Grafana datasource

6. **Tạo `grafana/dashboards/overview.json`**
   - Panel: RPS, P99 latency, active WebSocket, tasks/phút
   - Import tự động qua Grafana provisioning

---

## Phase 5 — Stripe Payment Integration

**Tại sao ấn tượng:** thay escrow giả bằng Stripe PaymentIntents thật — fintech/startup rất cần.  
**Files liên quan:** `backend/internal/service/escrow_service.go`, `backend/internal/domain/escrow.go`

### Bước thực hiện

1. **Thêm `github.com/stripe/stripe-go/v76` vào `go.mod`**

2. **Migration `000003_stripe_fields.up.sql`**
   - `ALTER TABLE escrow_transactions ADD COLUMN stripe_payment_intent_id TEXT`
   - `ALTER TABLE users ADD COLUMN stripe_customer_id TEXT`

3. **Tạo `backend/internal/service/stripe_escrow_service.go`**
   - `LockEscrow`: tạo PaymentIntent với `capture_method: manual` (hold funds)
   - `ReleaseEscrow`: `stripe.PaymentIntentCapture` (thu tiền thật)
   - `RefundEscrow`: `stripe.RefundNew` (hoàn tiền)
   - Implement interface `EscrowService` — không đổi code caller

4. **Thêm Stripe webhook endpoint `POST /webhooks/stripe`**
   - Verify signature với `STRIPE_WEBHOOK_SECRET`
   - Handle `payment_intent.succeeded`, `payment_intent.payment_failed`
   - Update escrow_transaction status theo event

5. **Thêm `STRIPE_SECRET_KEY` và `STRIPE_WEBHOOK_SECRET` vào env + docker-compose**

6. **Mobile: thêm màn hình nhập card**
   - Dùng `@stripe/stripe-react-native`
   - `PaymentSheet` trước khi tạo task (collect card info)

---

## Phase 6 — End-to-End Encrypted Chat (E2EE)

**Tại sao ấn tượng:** privacy-first, kỹ thuật nặng — ít project nào làm, rất nổi bật.  
**Files liên quan:** `mobile/services/api.ts`, `mobile/app/screens/ChatScreen.tsx`, `backend/internal/handler/chat_handler.go`

### Bước thực hiện

1. **Mobile: tạo `mobile/services/crypto.ts`**
   - Dùng `tweetnacl` (NaCl port for JS/RN)
   - Mỗi device tạo X25519 key pair khi khởi động, lưu vào SecureStore (Expo)
   - Public key gửi lên server khi register/init user

2. **Backend: migration + lưu public key**
   - `ALTER TABLE users ADD COLUMN public_key TEXT`
   - Endpoint `PUT /api/v1/users/me/pubkey` để update

3. **Key exchange khi mở chat**
   - Fetch public key của đối phương qua `GET /api/v1/users/{id}/pubkey`
   - Derive shared secret với `nacl.box.before(theirPublicKey, myPrivateKey)`

4. **Encrypt/Decrypt messages**
   - Send: `nacl.box(message, nonce, sharedKey)` → gửi `{ciphertext, nonce}` dạng base64
   - Receive: `nacl.box.open(ciphertext, nonce, sharedKey)`
   - Server chỉ lưu ciphertext, không đọc được nội dung

5. **UI indicator**
   - Badge "🔒 E2E Encrypted" ở header ChatScreen

---

## Phase 7 — Push Notifications (Firebase Cloud Messaging)

**Tại sao ấn tượng:** real-time UX — notify user ngay cả khi app không mở.  
**Files liên quan:** `backend/internal/service/claim_service.go`, `mobile/app/App.tsx`

### Bước thực hiện

1. **Firebase setup**
   - Tạo project Firebase, download `google-services.json` (Android) + `GoogleService-Info.plist` (iOS)
   - Thêm vào mobile app

2. **Mobile: `mobile/services/notifications.ts`**
   - Dùng `expo-notifications`
   - Request permission, get Expo push token
   - Gửi token lên `POST /api/v1/users/me/push-token`

3. **Backend: lưu push token**
   - `ALTER TABLE users ADD COLUMN push_token TEXT`
   - Endpoint `PUT /api/v1/users/me/push-token`

4. **Tạo `backend/internal/service/notification_service.go`**
   - `SendPushNotification(userID, title, body string)`
   - Gọi Firebase Admin SDK (`firebase.google.com/go/v4`)
   - Fallback: Expo Push API (đơn giản hơn, không cần native setup)

5. **Hook vào events**
   - `ClaimTask` → notify owner: "Có người nhận task của bạn"
   - `ApproveClaim` → notify claimer: "Task được duyệt, tiền sắp về"
   - `RejectClaim` → notify claimer: "Task bị từ chối"
   - `SubmitCompletion` → notify owner: "Claimer đã nộp bài"

---

## Phase 8 — Image Upload lên S3

**Tại sao ấn tượng:** presigned URL pattern — không expose AWS creds xuống client.  
**Files liên quan:** `backend/internal/handler/task_handler.go`, `mobile/app/screens/CreateTaskScreen.tsx`

### Bước thực hiện

1. **Thêm `github.com/aws/aws-sdk-go-v2` vào `go.mod`**

2. **Tạo `backend/internal/handler/upload_handler.go`**
   - `POST /api/v1/upload/presign` — nhận `{filename, content_type}`, trả về presigned PUT URL (15 phút TTL)
   - Client upload thẳng lên S3, không qua backend
   - Trả về `{upload_url, public_url}`

3. **S3 bucket config**
   - Bucket policy: public read cho `/task-images/*`
   - CORS config cho phép PUT từ mobile

4. **Migration `000004_image_url.up.sql`**
   - `ALTER TABLE tasks ADD COLUMN image_url TEXT`

5. **Mobile: Image picker trong `CreateTaskScreen`**
   - Dùng `expo-image-picker`
   - Pick ảnh → gọi `/upload/presign` → PUT lên S3 → lưu URL vào form
   - Preview ảnh trước khi submit
   - Hiển thị ảnh trong `TaskDetailScreen`

6. **Env: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_BUCKET_NAME`, `AWS_REGION`**

---

## Phase 9 — GitHub Actions CI/CD

**Tại sao ấn tượng:** DevOps hygiene — tự động test + build mỗi PR.  
**Files liên quan:** `.github/workflows/`

### Bước thực hiện

1. **Tạo `.github/workflows/backend-ci.yml`**
   - Trigger: push/PR vào `main`
   - Steps:
     - `go vet ./...`
     - `golangci-lint run`
     - `go test ./...` với PostgreSQL service container
     - `docker build` backend image

2. **Tạo `.github/workflows/mobile-ci.yml`**
   - Trigger: push/PR vào `main`
   - Steps:
     - `npm ci`
     - `npx tsc --noEmit` (type check)
     - `npx expo export --platform android` (bundle check)

3. **Tạo `.github/workflows/release.yml`**
   - Trigger: push tag `v*`
   - Build + push Docker image lên GHCR (GitHub Container Registry)
   - Tag image với version + `latest`

4. **Thêm badges vào README**
   - CI status badge
   - License badge

---

## Tổng hợp thứ tự làm

| # | Phase | Độ khó | Thời gian ước tính | CV Impact |
|---|-------|--------|-------------------|-----------|
| 1 | GitHub Actions CI/CD | Thấp | 2h | Trung bình |
| 2 | Per-User Rate Limiting | Thấp | 3h | Trung bình |
| 3 | Prometheus + Grafana | Trung bình | 4h | Cao |
| 4 | Full-Text Search | Trung bình | 4h | Cao |
| 5 | Push Notifications | Trung bình | 6h | Trung bình |
| 6 | Redis Pub/Sub WebSocket | Trung bình | 6h | Rất cao |
| 7 | Image Upload S3 | Trung bình | 6h | Trung bình |
| 8 | Stripe Payment | Cao | 10h | Rất cao |
| 9 | E2EE Chat | Cao | 12h | Rất cao |

**Đã làm xong:** CI/CD, rate limit, Prometheus, search, Redis WebSocket, push notifications, E2EE chat.  
**Còn lại đều cần credentials bên ngoài:** S3 (AWS), Stripe.

**Nếu chỉ có 1 tuần:** làm Phase 1–4 (CI/CD, Rate Limit, Prometheus, Search) — đủ để nói "production-grade backend" trong interview.

**Nếu có 2–3 tuần:** thêm Phase 6 (Redis WebSocket) + Phase 8 (Stripe) — stack CV cực mạnh cho Go/distributed systems roles.

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/repository"
)

// ExpoPushURL is the public Expo push service. It needs no credentials, which
// is why it is preferred here over the Firebase Admin SDK.
const ExpoPushURL = "https://exp.host/--/api/v2/push/send"

// pushCopy is the user-facing text for each event type. Events missing from
// this map are delivered over WebSocket only — not every in-app update is
// worth waking the phone for.
var pushCopy = map[string]struct{ Title, Body string }{
	"chat_message":         {"Tin nhắn mới", "Bạn có tin nhắn mới về task"},
	"claim_created":        {"Có người nhận task", "Task của bạn vừa được nhận"},
	"claim_approved":       {"Task được duyệt", "Tiền thưởng đang được chuyển cho bạn"},
	"claim_rejected":       {"Task bị từ chối", "Chủ task đã từ chối bài nộp của bạn"},
	"completion_submitted": {"Bài nộp mới", "Claimer đã nộp bài, hãy kiểm tra"},
}

// PushNotifier delivers events to a user's device through Expo. It implements
// Notifier, so it can be combined with the WebSocket hub via MultiNotifier.
type PushNotifier struct {
	users  repository.UserRepository
	url    string
	client *http.Client
}

func NewPushNotifier(users repository.UserRepository) *PushNotifier {
	// EXPO_PUSH_URL lets a test or staging environment point at a stub
	// instead of the real push service.
	url := os.Getenv("EXPO_PUSH_URL")
	if url == "" {
		url = ExpoPushURL
	}
	return &PushNotifier{
		users:  users,
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *PushNotifier) NotifyUser(userID uuid.UUID, eventType string, payload any) {
	text, ok := pushCopy[eventType]
	if !ok {
		return
	}

	// Fire and forget: a slow push service must never block the request that
	// triggered it.
	go func() {
		if err := p.send(userID, eventType, text.Title, text.Body, payload); err != nil {
			log.Printf("push notification (%s) failed: %v", eventType, err)
		}
	}()
}

func (p *PushNotifier) send(userID uuid.UUID, eventType, title, body string, payload any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := p.users.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}
	if user.PushToken == "" {
		return nil // user never registered a device
	}

	msg, err := json.Marshal(map[string]any{
		"to":    user.PushToken,
		"title": title,
		"body":  body,
		"sound": "default",
		"data":  map[string]any{"type": eventType, "payload": payload},
	})
	if err != nil {
		return fmt.Errorf("marshal push message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(msg))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("expo returned %s", resp.Status)
	}
	return nil
}

// MultiNotifier fans an event out to every notifier, so one event can reach
// both an open app (WebSocket) and a closed one (push).
type MultiNotifier []Notifier

func (m MultiNotifier) NotifyUser(userID uuid.UUID, eventType string, payload any) {
	for _, n := range m {
		n.NotifyUser(userID, eventType, payload)
	}
}

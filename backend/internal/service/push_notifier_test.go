package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/task-underground/backend/internal/domain"
)

// event is a notification captured by recordingNotifier.
type event struct {
	UserID uuid.UUID
	Type   string
}

// recordingNotifier captures notifications so tests can assert who was told
// what. Safe for concurrent use — push delivery happens on its own goroutine.
type recordingNotifier struct {
	mu   sync.Mutex
	seen []event
}

func (r *recordingNotifier) NotifyUser(userID uuid.UUID, eventType string, _ any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen = append(r.seen, event{UserID: userID, Type: eventType})
}

func (r *recordingNotifier) events() []event {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]event(nil), r.seen...)
}

// userRepoWithToken serves one user carrying a push token.
type userRepoWithToken struct {
	mockUserRepo
	token string
}

func (u *userRepoWithToken) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	return &domain.User{ID: id, PushToken: u.token}, nil
}

// newExpoStub stands in for the Expo push service and reports the bodies it
// receives.
func newExpoStub(t *testing.T) (*httptest.Server, chan map[string]any) {
	t.Helper()
	received := make(chan map[string]any, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		received <- body
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, received
}

func TestPushNotifierSendsToExpo(t *testing.T) {
	srv, received := newExpoStub(t)

	p := NewPushNotifier(&userRepoWithToken{token: "ExponentPushToken[abc123]"})
	p.url = srv.URL

	p.NotifyUser(uuid.New(), "claim_created", nil)

	select {
	case body := <-received:
		assert.Equal(t, "ExponentPushToken[abc123]", body["to"])
		assert.Equal(t, pushCopy["claim_created"].Title, body["title"])
		data := body["data"].(map[string]any)
		assert.Equal(t, "claim_created", data["type"])
	case <-time.After(2 * time.Second):
		t.Fatal("no push request reached Expo")
	}
}

func TestPushNotifierSkipsUserWithoutToken(t *testing.T) {
	srv, received := newExpoStub(t)

	p := NewPushNotifier(&userRepoWithToken{token: ""})
	p.url = srv.URL

	p.NotifyUser(uuid.New(), "claim_created", nil)

	select {
	case <-received:
		t.Fatal("sent a push to a user with no token")
	case <-time.After(300 * time.Millisecond):
	}
}

// Not every WebSocket event deserves a phone notification.
func TestPushNotifierIgnoresUnknownEvent(t *testing.T) {
	srv, received := newExpoStub(t)

	p := NewPushNotifier(&userRepoWithToken{token: "ExponentPushToken[abc123]"})
	p.url = srv.URL

	p.NotifyUser(uuid.New(), "some_internal_event", nil)

	select {
	case <-received:
		t.Fatal("sent a push for an event with no user-facing copy")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestMultiNotifierFansOut(t *testing.T) {
	a, b := &recordingNotifier{}, &recordingNotifier{}
	userID := uuid.New()

	MultiNotifier{a, b}.NotifyUser(userID, "chat_message", nil)

	want := []event{{UserID: userID, Type: "chat_message"}}
	assert.Equal(t, want, a.events())
	assert.Equal(t, want, b.events())
}

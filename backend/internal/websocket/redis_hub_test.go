package websocket

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/cache"
)

// Two hubs sharing one Redis stand in for two backend instances: a message
// published on one must reach a client connected to the other. miniredis runs
// in-process, so this covers the fanout without any external infrastructure.
func TestRedisFanoutCrossesInstances(t *testing.T) {
	server := miniredis.RunT(t)

	client := cache.NewClient("redis://" + server.Addr())
	if client == nil {
		t.Fatal("could not connect to miniredis")
	}
	t.Cleanup(func() { client.Close() })

	instanceA, instanceB := NewHub(), NewHub()
	instanceA.UseRedis(t.Context(), client)
	instanceB.UseRedis(t.Context(), client)

	// The subscriber goroutines need a moment before publishes are seen.
	time.Sleep(200 * time.Millisecond)

	userID := uuid.New()
	alice := newTestClient(instanceB, userID) // connected to instance B only

	instanceA.NotifyUser(userID, "claim_created", nil)

	select {
	case <-alice.Send:
	case <-time.After(2 * time.Second):
		t.Fatal("message published on instance A never reached instance B")
	}
}

// A message must be delivered exactly once even though the publishing
// instance also receives its own Redis message back.
func TestRedisFanoutDoesNotDuplicateOnPublisher(t *testing.T) {
	server := miniredis.RunT(t)

	client := cache.NewClient("redis://" + server.Addr())
	if client == nil {
		t.Fatal("could not connect to miniredis")
	}
	t.Cleanup(func() { client.Close() })

	hub := NewHub()
	hub.UseRedis(t.Context(), client)
	time.Sleep(200 * time.Millisecond)

	userID := uuid.New()
	alice := newTestClient(hub, userID)
	alice.Send = make(chan []byte, 4) // room to catch duplicates

	hub.NotifyUser(userID, "claim_created", nil)
	time.Sleep(300 * time.Millisecond)

	if got := len(alice.Send); got != 1 {
		t.Errorf("alice received %d copies, want exactly 1", got)
	}
}

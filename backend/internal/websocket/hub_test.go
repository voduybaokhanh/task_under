package websocket

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func newTestClient(hub *Hub, userID uuid.UUID) *Client {
	c := &Client{ID: uuid.New(), UserID: userID, Send: make(chan []byte, 1), Hub: hub}
	hub.clients[c.ID] = c
	return c
}

// Without Redis the hub must still deliver to locally connected clients, and
// only to the addressed users.
func TestSendWithoutRedisDeliversLocally(t *testing.T) {
	hub := NewHub()
	alice := newTestClient(hub, uuid.New())
	bob := newTestClient(hub, uuid.New())

	hub.BroadcastToUser(alice.UserID, Message{Type: "claim_created"})

	select {
	case data := <-alice.Send:
		var got Message
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got.Type != "claim_created" {
			t.Errorf("got type %q, want claim_created", got.Type)
		}
	default:
		t.Fatal("alice received nothing")
	}

	if len(bob.Send) != 0 {
		t.Error("bob received a message addressed to alice")
	}
}

// BroadcastToTask reaches every participant of the task.
func TestBroadcastToTaskReachesAllParticipants(t *testing.T) {
	hub := NewHub()
	owner := newTestClient(hub, uuid.New())
	claimer := newTestClient(hub, uuid.New())
	stranger := newTestClient(hub, uuid.New())

	hub.BroadcastToTask(uuid.New(), Message{Type: "chat_message"},
		[]uuid.UUID{owner.UserID, claimer.UserID})

	for name, c := range map[string]*Client{"owner": owner, "claimer": claimer} {
		if len(c.Send) != 1 {
			t.Errorf("%s received %d messages, want 1", name, len(c.Send))
		}
	}
	if len(stranger.Send) != 0 {
		t.Error("stranger received a task message")
	}
}

// The Redis envelope must survive a marshal/unmarshal round trip, since that
// is what crosses the Pub/Sub channel between instances.
func TestFanoutEnvelopeRoundTrip(t *testing.T) {
	hub := NewHub()
	alice := newTestClient(hub, uuid.New())

	payload, _ := json.Marshal(Message{Type: "task_completed"})
	raw, err := json.Marshal(fanoutEnvelope{UserIDs: []uuid.UUID{alice.UserID}, Data: payload})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var env fanoutEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	hub.deliver(env.UserIDs, env.Data)

	if len(alice.Send) != 1 {
		t.Fatal("alice received nothing from the fanout envelope")
	}
	if string(<-alice.Send) != string(payload) {
		t.Error("payload changed in transit")
	}
}

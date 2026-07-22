package websocket

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// fanoutChannel carries every cross-instance WebSocket delivery. A single
// channel is enough: each instance filters by user ID locally, and the
// number of instances is small compared to the number of users.
const fanoutChannel = "ws:fanout"

// fanoutEnvelope is what travels over Redis: the already-marshalled client
// payload plus the set of users it is addressed to.
type fanoutEnvelope struct {
	UserIDs []uuid.UUID     `json:"user_ids"`
	Data    json.RawMessage `json:"data"`
}

// UseRedis makes the hub broadcast through Redis Pub/Sub so that messages
// reach clients connected to any backend instance. Passing a nil client keeps
// the in-memory single-instance behaviour.
func (h *Hub) UseRedis(ctx context.Context, client *redis.Client) {
	if client == nil {
		return
	}
	h.redis = client
	go h.subscribe(ctx)
}

func (h *Hub) subscribe(ctx context.Context) {
	sub := h.redis.Subscribe(ctx, fanoutChannel)
	defer sub.Close()

	log.Printf("WebSocket hub subscribed to %s", fanoutChannel)

	for msg := range sub.Channel() {
		var env fanoutEnvelope
		if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
			log.Printf("Error unmarshaling fanout envelope: %v", err)
			continue
		}
		h.deliver(env.UserIDs, env.Data)
	}
}

// publish sends the payload to every instance (including this one, which
// receives it back through its own subscription — so callers must not also
// deliver locally). Returns false when Redis is not configured.
func (h *Hub) publish(userIDs []uuid.UUID, data []byte) bool {
	if h.redis == nil {
		return false
	}

	env, err := json.Marshal(fanoutEnvelope{UserIDs: userIDs, Data: data})
	if err != nil {
		log.Printf("Error marshaling fanout envelope: %v", err)
		return false
	}

	if err := h.redis.Publish(context.Background(), fanoutChannel, env).Err(); err != nil {
		log.Printf("Error publishing to Redis, falling back to local delivery: %v", err)
		return false
	}
	return true
}

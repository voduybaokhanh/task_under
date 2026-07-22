package cache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient builds a Redis client from a REDIS_URL (redis://host:port[/db]).
// It returns nil when redisURL is empty or the server is unreachable, so callers
// can gracefully fall back to in-memory / single-instance behaviour.
func NewClient(redisURL string) *redis.Client {
	if redisURL == "" {
		log.Println("REDIS_URL not set; running without Redis (single-instance mode)")
		return nil
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Invalid REDIS_URL (%v); running without Redis", err)
		return nil
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Redis unreachable (%v); running without Redis", err)
		_ = client.Close()
		return nil
	}

	log.Println("Connected to Redis")
	return client
}

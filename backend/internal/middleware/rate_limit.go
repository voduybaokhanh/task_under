package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// PerUserRateLimit returns a Gin middleware implementing a per-device sliding
// window rate limiter backed by Redis sorted sets.
//
// For each request it:
//  1. Removes entries older than the window (ZREMRANGEBYSCORE).
//  2. Counts the remaining entries in the window (ZCARD).
//  3. Rejects with 429 if the limit is exceeded, otherwise records the request.
//
// The window is enforced per device (X-Device-ID header), so different devices
// never affect each other's budget.
//
// If rdb is nil (Redis not configured) the middleware is a no-op, keeping local
// single-instance development working without Redis.
func PerUserRateLimit(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	if rdb == nil {
		return func(c *gin.Context) { c.Next() }
	}

	windowMs := window.Milliseconds()

	return func(c *gin.Context) {
		deviceID := c.GetHeader("X-Device-ID")
		if deviceID == "" {
			// No identity yet; let the auth layer reject it.
			c.Next()
			return
		}

		ctx := c.Request.Context()
		key := "rate:" + deviceID
		now := time.Now().UnixMilli()
		windowStart := now - windowMs

		// Drop entries outside the window, then count what's left.
		pipe := rdb.Pipeline()
		pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10))
		countCmd := pipe.ZCard(ctx, key)
		if _, err := pipe.Exec(ctx); err != nil {
			// Fail open: don't take the API down if Redis hiccups.
			c.Next()
			return
		}

		count := int(countCmd.Val())
		remaining := limit - count
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))

		if count >= limit {
			retryAfter := int(window.Seconds())
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			c.Abort()
			return
		}

		// Record this request; unique member so concurrent requests don't collide.
		member := strconv.FormatInt(now, 10) + "-" + uuid.NewString()
		addPipe := rdb.Pipeline()
		addPipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: member})
		addPipe.Expire(ctx, key, window)
		_, _ = addPipe.Exec(ctx)

		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining-1))
		c.Next()
	}
}

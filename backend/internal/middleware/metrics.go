package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/task-underground/backend/internal/metrics"
)

// Metrics records request count and latency for every request. It uses the
// matched route template (e.g. /api/v1/task/:id) as the path label to keep
// cardinality bounded regardless of dynamic IDs.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		elapsed := time.Since(start).Seconds()

		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, path, status).Observe(elapsed)
	}
}

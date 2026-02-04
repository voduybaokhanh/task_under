package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/service"
)

const UserIDKey = "user_id"

func AuthMiddleware(userSvc service.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceID := c.GetHeader("X-Device-ID")
		if deviceID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "X-Device-ID header required"})
			c.Abort()
			return
		}

		user, err := userSvc.GetOrCreateUser(c.Request.Context(), deviceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to authenticate"})
			c.Abort()
			return
		}

		c.Set(UserIDKey, user.ID)
		c.Set("user", user)
		c.Next()
	}
}

func GetUserID(c *gin.Context) uuid.UUID {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return uuid.Nil
	}
	return userID.(uuid.UUID)
}

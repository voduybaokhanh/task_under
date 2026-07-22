package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/task-underground/backend/internal/domain"
	"github.com/task-underground/backend/internal/middleware"
	"github.com/task-underground/backend/internal/service"
)

type UserHandler struct {
	userSvc service.UserService
}

func NewUserHandler(userSvc service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	c.JSON(http.StatusOK, user.(*domain.User))
}

type updatePushTokenRequest struct {
	PushToken string `json:"push_token" binding:"required"`
}

// UpdatePushToken stores the Expo push token of the caller's device so the
// backend can reach them while the app is closed.
func (h *UserHandler) UpdatePushToken(c *gin.Context) {
	var req updatePushTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userSvc.UpdatePushToken(c.Request.Context(), middleware.GetUserID(c), req.PushToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "push token updated"})
}

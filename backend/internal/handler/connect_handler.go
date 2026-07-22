package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/task-underground/backend/internal/middleware"
	"github.com/task-underground/backend/internal/service"
)

type ConnectHandler struct {
	connect service.ConnectService
}

func NewConnectHandler(connect service.ConnectService) *ConnectHandler {
	return &ConnectHandler{connect: connect}
}

// StartOnboarding returns the Stripe-hosted URL where a claimer sets up the
// account their earnings are paid into.
func (h *ConnectHandler) StartOnboarding(c *gin.Context) {
	if h.connect == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payouts are not configured"})
		return
	}

	url, err := h.connect.OnboardingLink(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"onboarding_url": url})
}

// PayoutStatus tells the app whether this user can be paid yet.
func (h *ConnectHandler) PayoutStatus(c *gin.Context) {
	if h.connect == nil {
		c.JSON(http.StatusOK, gin.H{"payouts_enabled": false, "configured": false})
		return
	}

	enabled, err := h.connect.PayoutsEnabled(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"payouts_enabled": enabled, "configured": true})
}

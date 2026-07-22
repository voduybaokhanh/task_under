package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/task-underground/backend/internal/middleware"
	"github.com/task-underground/backend/internal/service"
)

type PaymentHandler struct {
	payments service.PaymentIntentProvider
}

func NewPaymentHandler(payments service.PaymentIntentProvider) *PaymentHandler {
	return &PaymentHandler{payments: payments}
}

// GetClientSecret hands the task owner the secret their app needs to attach a
// card to the escrow hold. Nobody else may have it — it authorises a charge.
func (h *PaymentHandler) GetClientSecret(c *gin.Context) {
	if h.payments == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "card payments are not configured"})
		return
	}

	secret, err := h.payments.ClientSecret(c.Request.Context(), parseUUID(c.Param("tid")), middleware.GetUserID(c))
	if err != nil {
		if err == service.ErrUnauthorized {
			c.JSON(http.StatusForbidden, gin.H{"error": "only the task owner can pay for it"})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"client_secret": secret})
}

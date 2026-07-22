package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"github.com/task-underground/backend/internal/domain"
	"github.com/task-underground/backend/internal/repository"
)

// maxWebhookBody caps what we read from the webhook endpoint — it is public.
const maxWebhookBody = 64 << 10

type StripeWebhookHandler struct {
	escrowRepo repository.EscrowRepository
	secret     string
}

func NewStripeWebhookHandler(escrowRepo repository.EscrowRepository) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		escrowRepo: escrowRepo,
		secret:     os.Getenv("STRIPE_WEBHOOK_SECRET"),
	}
}

// Handle records what Stripe tells us about a payment. Every request is
// signature-checked: this endpoint is unauthenticated, so the signature is the
// only thing separating Stripe from anyone who knows the URL.
func (h *StripeWebhookHandler) Handle(c *gin.Context) {
	if h.secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "stripe webhooks are not configured"})
		return
	}

	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBody))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	// The signature is still verified in full; only the API-version check is
	// relaxed, because the account's version is a dashboard setting we do not
	// control and we read just the event type and the payment intent ID —
	// fields that are stable across versions.
	event, err := webhook.ConstructEventWithOptions(payload, c.GetHeader("Stripe-Signature"), h.secret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true})
	if err != nil {
		log.Printf("Rejected Stripe webhook: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	var status domain.EscrowTransactionStatus
	switch event.Type {
	case "payment_intent.succeeded", "payment_intent.amount_capturable_updated":
		status = domain.EscrowStatusCompleted
	case "payment_intent.payment_failed", "payment_intent.canceled":
		status = domain.EscrowStatusFailed
	default:
		// Stripe retries anything we do not 2xx, so acknowledge the rest.
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	var intent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &intent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot parse payment intent"})
		return
	}

	tx, err := h.escrowRepo.GetByStripePaymentIntent(c.Request.Context(), intent.ID)
	if err != nil {
		// An intent we do not know about is not an error worth retrying.
		log.Printf("Stripe webhook for unknown payment intent %s", intent.ID)
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	if err := h.escrowRepo.UpdateTransactionStatus(c.Request.Context(), tx.ID, status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Stripe %s → escrow %s is %s", event.Type, tx.ID, status)
	c.JSON(http.StatusOK, gin.H{"received": true})
}

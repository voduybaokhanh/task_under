package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/task-underground/backend/internal/domain"
)

const testWebhookSecret = "whsec_test_secret"

// escrowRepoStub records status changes so tests can assert on them.
type escrowRepoStub struct {
	tx      *domain.EscrowTransaction
	updated domain.EscrowTransactionStatus
}

func (r *escrowRepoStub) CreateTransaction(context.Context, *domain.EscrowTransaction) error {
	return nil
}

func (r *escrowRepoStub) GetTransactionsByTaskID(context.Context, uuid.UUID) ([]*domain.EscrowTransaction, error) {
	return nil, nil
}

func (r *escrowRepoStub) UpdateTransactionStatus(_ context.Context, _ uuid.UUID, status domain.EscrowTransactionStatus) error {
	r.updated = status
	return nil
}

func (r *escrowRepoStub) SetStripePaymentIntent(context.Context, uuid.UUID, string) error {
	return nil
}

func (r *escrowRepoStub) GetByStripePaymentIntent(_ context.Context, id string) (*domain.EscrowTransaction, error) {
	if r.tx != nil && r.tx.StripePaymentIntentID == id {
		return r.tx, nil
	}
	return nil, sql.ErrNoRows
}

// sign builds the Stripe-Signature header the way Stripe does, so the handler
// is tested through real signature verification rather than a bypass.
func sign(payload string, secret string, at time.Time) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", at.Unix(), payload)
	return fmt.Sprintf("t=%d,v1=%s", at.Unix(), hex.EncodeToString(mac.Sum(nil)))
}

func postWebhook(t *testing.T, repo *escrowRepoStub, payload, signature string) *httptest.ResponseRecorder {
	t.Helper()
	t.Setenv("STRIPE_WEBHOOK_SECRET", testWebhookSecret)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/webhooks/stripe", NewStripeWebhookHandler(repo).Handle)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader(payload))
	req.Header.Set("Stripe-Signature", signature)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func eventPayload(eventType, intentID string) string {
	return fmt.Sprintf(`{"id":"evt_1","type":%q,"data":{"object":{"id":%q,"object":"payment_intent"}}}`,
		eventType, intentID)
}

func TestWebhookMarksPaymentSucceeded(t *testing.T) {
	repo := &escrowRepoStub{tx: &domain.EscrowTransaction{
		ID: uuid.New(), StripePaymentIntentID: "pi_1", Status: domain.EscrowStatusPending,
	}}
	payload := eventPayload("payment_intent.succeeded", "pi_1")

	rec := postWebhook(t, repo, payload, sign(payload, testWebhookSecret, time.Now()))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, domain.EscrowStatusCompleted, repo.updated)
}

func TestWebhookMarksPaymentFailed(t *testing.T) {
	repo := &escrowRepoStub{tx: &domain.EscrowTransaction{
		ID: uuid.New(), StripePaymentIntentID: "pi_1", Status: domain.EscrowStatusPending,
	}}
	payload := eventPayload("payment_intent.payment_failed", "pi_1")

	rec := postWebhook(t, repo, payload, sign(payload, testWebhookSecret, time.Now()))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, domain.EscrowStatusFailed, repo.updated)
}

// The endpoint is public, so a forged event must change nothing.
func TestWebhookRejectsABadSignature(t *testing.T) {
	repo := &escrowRepoStub{tx: &domain.EscrowTransaction{
		ID: uuid.New(), StripePaymentIntentID: "pi_1", Status: domain.EscrowStatusPending,
	}}
	payload := eventPayload("payment_intent.succeeded", "pi_1")

	rec := postWebhook(t, repo, payload, sign(payload, "whsec_wrong_secret", time.Now()))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, repo.updated, "an unverified event must not touch the escrow")
}

// A valid signature captured hours ago must not be replayable.
func TestWebhookRejectsAnOldSignature(t *testing.T) {
	repo := &escrowRepoStub{tx: &domain.EscrowTransaction{
		ID: uuid.New(), StripePaymentIntentID: "pi_1", Status: domain.EscrowStatusPending,
	}}
	payload := eventPayload("payment_intent.succeeded", "pi_1")

	rec := postWebhook(t, repo, payload, sign(payload, testWebhookSecret, time.Now().Add(-2*time.Hour)))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Empty(t, repo.updated)
}

// Events we do not act on must still be acknowledged, or Stripe retries them
// forever.
func TestWebhookAcknowledgesUnhandledEvents(t *testing.T) {
	repo := &escrowRepoStub{}
	payload := eventPayload("customer.created", "pi_1")

	rec := postWebhook(t, repo, payload, sign(payload, testWebhookSecret, time.Now()))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, repo.updated)
}

func TestWebhookIsUnavailableWithoutASecret(t *testing.T) {
	t.Setenv("STRIPE_WEBHOOK_SECRET", "")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/webhooks/stripe", NewStripeWebhookHandler(&escrowRepoStub{}).Handle)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/stripe", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

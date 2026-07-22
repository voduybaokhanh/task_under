package domain

import (
	"time"

	"github.com/google/uuid"
)

type EscrowTransactionType string

const (
	EscrowTypeLock    EscrowTransactionType = "lock"
	EscrowTypeRelease EscrowTransactionType = "release"
	EscrowTypeRefund  EscrowTransactionType = "refund"
	// EscrowTypePayout is the transfer of a captured payment to the claimer's
	// own Stripe Connect account.
	EscrowTypePayout EscrowTransactionType = "payout"
)

type EscrowTransactionStatus string

const (
	EscrowStatusPending   EscrowTransactionStatus = "pending"
	EscrowStatusCompleted EscrowTransactionStatus = "completed"
	EscrowStatusFailed    EscrowTransactionStatus = "failed"
)

type EscrowTransaction struct {
	ID              uuid.UUID               `json:"id"`
	TaskID          uuid.UUID               `json:"task_id"`
	UserID          uuid.UUID               `json:"user_id"`
	Amount          float64                 `json:"amount"`
	TransactionType EscrowTransactionType   `json:"transaction_type"`
	Status          EscrowTransactionStatus `json:"status"`
	// StripePaymentIntentID is empty when escrow is simulated (no Stripe key).
	StripePaymentIntentID string     `json:"stripe_payment_intent_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	CompletedAt           *time.Time `json:"completed_at,omitempty"`
}

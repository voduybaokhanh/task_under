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
)

type EscrowTransactionStatus string

const (
	EscrowStatusPending   EscrowTransactionStatus = "pending"
	EscrowStatusCompleted EscrowTransactionStatus = "completed"
	EscrowStatusFailed    EscrowTransactionStatus = "failed"
)

type EscrowTransaction struct {
	ID              uuid.UUID            `json:"id"`
	TaskID          uuid.UUID            `json:"task_id"`
	UserID          uuid.UUID            `json:"user_id"`
	Amount          float64              `json:"amount"`
	TransactionType EscrowTransactionType `json:"transaction_type"`
	Status          EscrowTransactionStatus `json:"status"`
	CreatedAt       time.Time            `json:"created_at"`
	CompletedAt     *time.Time           `json:"completed_at,omitempty"`
}

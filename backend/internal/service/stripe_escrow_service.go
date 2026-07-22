package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"github.com/task-underground/backend/internal/domain"
	"github.com/task-underground/backend/internal/repository"
)

// ErrNoPaymentIntent means the task's money was never held by Stripe, so there
// is nothing to capture or refund.
var ErrNoPaymentIntent = errors.New("no stripe payment intent for this task")

// StripeAPI is the slice of Stripe we use. An interface keeps the tests off
// the network and makes the SDK swappable.
type StripeAPI interface {
	CreatePaymentIntent(ctx context.Context, params *stripe.PaymentIntentParams) (*stripe.PaymentIntent, error)
	CapturePaymentIntent(ctx context.Context, id string, params *stripe.PaymentIntentCaptureParams) (*stripe.PaymentIntent, error)
	CancelPaymentIntent(ctx context.Context, id string, params *stripe.PaymentIntentCancelParams) (*stripe.PaymentIntent, error)
	CreateRefund(ctx context.Context, params *stripe.RefundParams) (*stripe.Refund, error)
	GetPaymentIntent(ctx context.Context, id string) (*stripe.PaymentIntent, error)
}

// PaymentIntentProvider exposes the client secret a mobile app needs to attach
// a card to the hold. Only the Stripe-backed escrow implements it.
type PaymentIntentProvider interface {
	ClientSecret(ctx context.Context, taskID, ownerID uuid.UUID) (string, error)
}

// stripeEscrowService implements EscrowService against real card payments:
// locking authorises the card (manual capture), releasing captures it, and
// refunding gives the money back.
type stripeEscrowService struct {
	escrowRepo repository.EscrowRepository
	taskRepo   repository.TaskRepository
	stripe     StripeAPI
	currency   string
}

func NewStripeEscrowService(
	escrowRepo repository.EscrowRepository,
	taskRepo repository.TaskRepository,
	client StripeAPI,
	currency string,
) EscrowService {
	if currency == "" {
		currency = string(stripe.CurrencyUSD)
	}
	return &stripeEscrowService{
		escrowRepo: escrowRepo,
		taskRepo:   taskRepo,
		stripe:     client,
		currency:   currency,
	}
}

// toMinorUnits converts a reward to the integer amount Stripe expects (cents).
func toMinorUnits(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

// LockEscrow authorises the owner's card without taking the money: the funds
// are held until the task is approved or cancelled.
func (s *stripeEscrowService) LockEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.EscrowLocked {
		return ErrEscrowAlreadyLocked
	}

	tx := &domain.EscrowTransaction{
		ID:              uuid.New(),
		TaskID:          taskID,
		UserID:          userID,
		Amount:          amount,
		TransactionType: domain.EscrowTypeLock,
		Status:          domain.EscrowStatusPending,
	}
	if err := s.escrowRepo.CreateTransaction(ctx, tx); err != nil {
		return err
	}

	intent, err := s.stripe.CreatePaymentIntent(ctx, &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(toMinorUnits(amount)),
		Currency:      stripe.String(s.currency),
		CaptureMethod: stripe.String(string(stripe.PaymentIntentCaptureMethodManual)),
		Metadata: map[string]string{
			"task_id":   taskID.String(),
			"user_id":   userID.String(),
			"escrow_id": tx.ID.String(),
		},
	})
	if err != nil {
		// Leave a failed row behind: a payment that never happened should be
		// visible, not silently missing.
		_ = s.escrowRepo.UpdateTransactionStatus(ctx, tx.ID, domain.EscrowStatusFailed)
		return fmt.Errorf("create payment intent: %w", err)
	}

	if err := s.escrowRepo.SetStripePaymentIntent(ctx, tx.ID, intent.ID); err != nil {
		return err
	}
	if err := s.taskRepo.SetEscrowLocked(ctx, taskID, true); err != nil {
		return err
	}

	// The hold exists, but the money only moves on capture — the escrow row
	// stays pending until the webhook or the capture confirms it.
	return nil
}

// ReleaseEscrow captures the authorised amount, paying the claimer.
func (s *stripeEscrowService) ReleaseEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error {
	lock, err := s.findLock(ctx, taskID)
	if err != nil {
		return err
	}

	tx := &domain.EscrowTransaction{
		ID:                    uuid.New(),
		TaskID:                taskID,
		UserID:                userID,
		Amount:                amount,
		TransactionType:       domain.EscrowTypeRelease,
		Status:                domain.EscrowStatusPending,
		StripePaymentIntentID: lock.StripePaymentIntentID,
	}
	if err := s.escrowRepo.CreateTransaction(ctx, tx); err != nil {
		return err
	}

	_, err = s.stripe.CapturePaymentIntent(ctx, lock.StripePaymentIntentID, &stripe.PaymentIntentCaptureParams{
		AmountToCapture: stripe.Int64(toMinorUnits(amount)),
	})
	if err != nil {
		_ = s.escrowRepo.UpdateTransactionStatus(ctx, tx.ID, domain.EscrowStatusFailed)
		return fmt.Errorf("capture payment intent: %w", err)
	}

	if err := s.escrowRepo.UpdateTransactionStatus(ctx, lock.ID, domain.EscrowStatusCompleted); err != nil {
		return err
	}
	return s.escrowRepo.UpdateTransactionStatus(ctx, tx.ID, domain.EscrowStatusCompleted)
}

// RefundEscrow returns the money to the owner. An uncaptured hold is cancelled
// rather than refunded — Stripe has not taken anything yet.
func (s *stripeEscrowService) RefundEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error {
	lock, err := s.findLock(ctx, taskID)
	if err != nil {
		return err
	}

	tx := &domain.EscrowTransaction{
		ID:                    uuid.New(),
		TaskID:                taskID,
		UserID:                userID,
		Amount:                amount,
		TransactionType:       domain.EscrowTypeRefund,
		Status:                domain.EscrowStatusPending,
		StripePaymentIntentID: lock.StripePaymentIntentID,
	}
	if err := s.escrowRepo.CreateTransaction(ctx, tx); err != nil {
		return err
	}

	if lock.Status == domain.EscrowStatusCompleted {
		_, err = s.stripe.CreateRefund(ctx, &stripe.RefundParams{
			PaymentIntent: stripe.String(lock.StripePaymentIntentID),
			Amount:        stripe.Int64(toMinorUnits(amount)),
		})
	} else {
		_, err = s.stripe.CancelPaymentIntent(ctx, lock.StripePaymentIntentID, &stripe.PaymentIntentCancelParams{})
	}
	if err != nil {
		_ = s.escrowRepo.UpdateTransactionStatus(ctx, tx.ID, domain.EscrowStatusFailed)
		return fmt.Errorf("refund payment intent: %w", err)
	}

	if err := s.taskRepo.SetEscrowLocked(ctx, taskID, false); err != nil {
		return err
	}
	return s.escrowRepo.UpdateTransactionStatus(ctx, tx.ID, domain.EscrowStatusCompleted)
}

// findLock returns the lock transaction holding this task's payment intent.
func (s *stripeEscrowService) findLock(ctx context.Context, taskID uuid.UUID) (*domain.EscrowTransaction, error) {
	txs, err := s.escrowRepo.GetTransactionsByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	for _, tx := range txs {
		if tx.TransactionType == domain.EscrowTypeLock && tx.StripePaymentIntentID != "" {
			return tx, nil
		}
	}
	return nil, ErrNoPaymentIntent
}

// ClientSecret returns the secret the task owner's app uses to present a
// payment sheet and attach a card to the existing hold. Only the owner may
// fetch it: it authorises charging that payment intent.
func (s *stripeEscrowService) ClientSecret(ctx context.Context, taskID, ownerID uuid.UUID) (string, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return "", err
	}
	if task.OwnerID != ownerID {
		return "", ErrUnauthorized
	}

	lock, err := s.findLock(ctx, taskID)
	if err != nil {
		return "", err
	}

	intent, err := s.stripe.GetPaymentIntent(ctx, lock.StripePaymentIntentID)
	if err != nil {
		return "", fmt.Errorf("retrieve payment intent: %w", err)
	}
	return intent.ClientSecret, nil
}

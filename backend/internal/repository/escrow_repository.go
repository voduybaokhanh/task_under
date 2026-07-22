package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
)

type EscrowRepository interface {
	CreateTransaction(ctx context.Context, tx *domain.EscrowTransaction) error
	GetTransactionsByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.EscrowTransaction, error)
	UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status domain.EscrowTransactionStatus) error
	SetStripePaymentIntent(ctx context.Context, id uuid.UUID, paymentIntentID string) error
	GetByStripePaymentIntent(ctx context.Context, paymentIntentID string) (*domain.EscrowTransaction, error)
}

type escrowRepository struct {
	db *sql.DB
}

func NewEscrowRepository(db *sql.DB) EscrowRepository {
	return &escrowRepository{db: db}
}

func (r *escrowRepository) CreateTransaction(ctx context.Context, tx *domain.EscrowTransaction) error {
	const q = `
		INSERT INTO escrow_transactions (id, task_id, user_id, amount, transaction_type, status, stripe_payment_intent_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7)
		RETURNING created_at
	`
	return r.db.QueryRowContext(ctx, q,
		tx.ID.String(), tx.TaskID.String(), tx.UserID.String(),
		tx.Amount, tx.TransactionType, tx.Status, tx.StripePaymentIntentID,
	).Scan(&tx.CreatedAt)
}

func (r *escrowRepository) GetTransactionsByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.EscrowTransaction, error) {
	const q = `
		SELECT id, task_id, user_id, amount, transaction_type, status, stripe_payment_intent_id, created_at, completed_at
		FROM escrow_transactions WHERE task_id = $1::uuid ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, taskID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*domain.EscrowTransaction
	for rows.Next() {
		tx := &domain.EscrowTransaction{}
		var completedAt sql.NullTime
		err := rows.Scan(&tx.ID, &tx.TaskID, &tx.UserID, &tx.Amount, &tx.TransactionType, &tx.Status,
			&tx.StripePaymentIntentID, &tx.CreatedAt, &completedAt)
		if err != nil {
			return nil, err
		}
		if completedAt.Valid {
			tx.CompletedAt = &completedAt.Time
		}
		txs = append(txs, tx)
	}
	return txs, rows.Err()
}

func (r *escrowRepository) UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status domain.EscrowTransactionStatus) error {
	const q = `
		UPDATE escrow_transactions
		SET status = $1, completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END
		WHERE id = $3::uuid
	`
	_, err := r.db.ExecContext(ctx, q, status, string(status), id.String())
	return err
}

func (r *escrowRepository) SetStripePaymentIntent(ctx context.Context, id uuid.UUID, paymentIntentID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE escrow_transactions SET stripe_payment_intent_id = $1 WHERE id = $2::uuid`,
		paymentIntentID, id.String())
	return err
}

func (r *escrowRepository) GetByStripePaymentIntent(ctx context.Context, paymentIntentID string) (*domain.EscrowTransaction, error) {
	const q = `
		SELECT id, task_id, user_id, amount, transaction_type, status, stripe_payment_intent_id, created_at, completed_at
		FROM escrow_transactions WHERE stripe_payment_intent_id = $1
		ORDER BY created_at DESC LIMIT 1
	`
	tx := &domain.EscrowTransaction{}
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, q, paymentIntentID).Scan(
		&tx.ID, &tx.TaskID, &tx.UserID, &tx.Amount, &tx.TransactionType, &tx.Status,
		&tx.StripePaymentIntentID, &tx.CreatedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		tx.CompletedAt = &completedAt.Time
	}
	return tx, nil
}

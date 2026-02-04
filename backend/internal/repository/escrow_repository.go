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
}

type escrowRepository struct {
	db *sql.DB
}

func NewEscrowRepository(db *sql.DB) EscrowRepository {
	return &escrowRepository{db: db}
}

func (r *escrowRepository) CreateTransaction(ctx context.Context, tx *domain.EscrowTransaction) error {
	query := `
		INSERT INTO escrow_transactions (id, task_id, user_id, amount, transaction_type, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`
	
	err := r.db.QueryRowContext(ctx, query,
		tx.ID,
		tx.TaskID,
		tx.UserID,
		tx.Amount,
		tx.TransactionType,
		tx.Status,
	).Scan(&tx.CreatedAt)
	
	return err
}

func (r *escrowRepository) GetTransactionsByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.EscrowTransaction, error) {
	query := `
		SELECT id, task_id, user_id, amount, transaction_type, status, created_at, completed_at
		FROM escrow_transactions
		WHERE task_id = $1
		ORDER BY created_at ASC
	`
	
	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var transactions []*domain.EscrowTransaction
	for rows.Next() {
		tx := &domain.EscrowTransaction{}
		var completedAt sql.NullTime
		err := rows.Scan(
			&tx.ID,
			&tx.TaskID,
			&tx.UserID,
			&tx.Amount,
			&tx.TransactionType,
			&tx.Status,
			&tx.CreatedAt,
			&completedAt,
		)
		if err != nil {
			return nil, err
		}
		if completedAt.Valid {
			tx.CompletedAt = &completedAt.Time
		}
		transactions = append(transactions, tx)
	}
	return transactions, rows.Err()
}

func (r *escrowRepository) UpdateTransactionStatus(ctx context.Context, id uuid.UUID, status domain.EscrowTransactionStatus) error {
	query := `
		UPDATE escrow_transactions 
		SET status = $1, completed_at = CASE WHEN $1 = 'completed' THEN NOW() ELSE completed_at END
		WHERE id = $2
	`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

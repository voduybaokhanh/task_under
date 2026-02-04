package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
)

type ClaimRepository interface {
	Create(ctx context.Context, claim *domain.Claim) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Claim, error)
	GetByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.Claim, error)
	GetByTaskIDAndClaimerID(ctx context.Context, taskID, claimerID uuid.UUID) (*domain.Claim, error)
	CountByTaskID(ctx context.Context, taskID uuid.UUID) (int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ClaimStatus) error
	SubmitCompletion(ctx context.Context, id uuid.UUID, text, imageURL string) error
}

type claimRepository struct {
	db *sql.DB
}

func NewClaimRepository(db *sql.DB) ClaimRepository {
	return &claimRepository{db: db}
}

func (r *claimRepository) Create(ctx context.Context, claim *domain.Claim) error {
	query := `
		INSERT INTO claims (id, task_id, claimer_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`
	
	err := r.db.QueryRowContext(ctx, query,
		claim.ID,
		claim.TaskID,
		claim.ClaimerID,
		claim.Status,
	).Scan(&claim.CreatedAt, &claim.UpdatedAt)
	
	return err
}

func (r *claimRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Claim, error) {
	query := `
		SELECT id, task_id, claimer_id, status, submitted_at, completion_text, completion_image_url, created_at, updated_at
		FROM claims
		WHERE id = $1
	`
	
	claim := &domain.Claim{}
	var submittedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&claim.ID,
		&claim.TaskID,
		&claim.ClaimerID,
		&claim.Status,
		&submittedAt,
		&claim.CompletionText,
		&claim.CompletionImageURL,
		&claim.CreatedAt,
		&claim.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if submittedAt.Valid {
		claim.SubmittedAt = &submittedAt.Time
	}
	return claim, nil
}

func (r *claimRepository) GetByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.Claim, error) {
	query := `
		SELECT id, task_id, claimer_id, status, submitted_at, completion_text, completion_image_url, created_at, updated_at
		FROM claims
		WHERE task_id = $1
		ORDER BY created_at ASC
	`
	
	rows, err := r.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var claims []*domain.Claim
	for rows.Next() {
		claim := &domain.Claim{}
		var submittedAt sql.NullTime
		err := rows.Scan(
			&claim.ID,
			&claim.TaskID,
			&claim.ClaimerID,
			&claim.Status,
			&submittedAt,
			&claim.CompletionText,
			&claim.CompletionImageURL,
			&claim.CreatedAt,
			&claim.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if submittedAt.Valid {
			claim.SubmittedAt = &submittedAt.Time
		}
		claims = append(claims, claim)
	}
	return claims, rows.Err()
}

func (r *claimRepository) GetByTaskIDAndClaimerID(ctx context.Context, taskID, claimerID uuid.UUID) (*domain.Claim, error) {
	query := `
		SELECT id, task_id, claimer_id, status, submitted_at, completion_text, completion_image_url, created_at, updated_at
		FROM claims
		WHERE task_id = $1 AND claimer_id = $2
	`
	
	claim := &domain.Claim{}
	var submittedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, taskID, claimerID).Scan(
		&claim.ID,
		&claim.TaskID,
		&claim.ClaimerID,
		&claim.Status,
		&submittedAt,
		&claim.CompletionText,
		&claim.CompletionImageURL,
		&claim.CreatedAt,
		&claim.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if submittedAt.Valid {
		claim.SubmittedAt = &submittedAt.Time
	}
	return claim, nil
}

func (r *claimRepository) CountByTaskID(ctx context.Context, taskID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM claims WHERE task_id = $1 AND status != 'cancelled'`
	var count int
	err := r.db.QueryRowContext(ctx, query, taskID).Scan(&count)
	return count, err
}

func (r *claimRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ClaimStatus) error {
	query := `UPDATE claims SET status = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *claimRepository) SubmitCompletion(ctx context.Context, id uuid.UUID, text, imageURL string) error {
	query := `
		UPDATE claims 
		SET completion_text = $1, completion_image_url = $2, submitted_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, text, imageURL, id)
	return err
}

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

func scanClaim(row interface{ Scan(...interface{}) error }) (*domain.Claim, error) {
	claim := &domain.Claim{}
	var submittedAt sql.NullTime
	var completionText, completionImageURL sql.NullString
	err := row.Scan(
		&claim.ID, &claim.TaskID, &claim.ClaimerID, &claim.Status,
		&submittedAt, &completionText, &completionImageURL,
		&claim.CreatedAt, &claim.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if submittedAt.Valid {
		claim.SubmittedAt = &submittedAt.Time
	}
	claim.CompletionText = completionText.String
	claim.CompletionImageURL = completionImageURL.String
	return claim, nil
}

func (r *claimRepository) Create(ctx context.Context, claim *domain.Claim) error {
	const q = `
		INSERT INTO claims (id, task_id, claimer_id, status)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, q,
		claim.ID.String(), claim.TaskID.String(), claim.ClaimerID.String(), claim.Status,
	).Scan(&claim.CreatedAt, &claim.UpdatedAt)
}

func (r *claimRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Claim, error) {
	const q = `
		SELECT id, task_id, claimer_id, status, submitted_at, completion_text, completion_image_url, created_at, updated_at
		FROM claims WHERE id = $1::uuid
	`
	return scanClaim(r.db.QueryRowContext(ctx, q, id.String()))
}

func (r *claimRepository) GetByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.Claim, error) {
	const q = `
		SELECT id, task_id, claimer_id, status, submitted_at, completion_text, completion_image_url, created_at, updated_at
		FROM claims WHERE task_id = $1::uuid ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q, taskID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var claims []*domain.Claim
	for rows.Next() {
		claim, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		claims = append(claims, claim)
	}
	return claims, rows.Err()
}

func (r *claimRepository) GetByTaskIDAndClaimerID(ctx context.Context, taskID, claimerID uuid.UUID) (*domain.Claim, error) {
	const q = `
		SELECT id, task_id, claimer_id, status, submitted_at, completion_text, completion_image_url, created_at, updated_at
		FROM claims WHERE task_id = $1::uuid AND claimer_id = $2::uuid
	`
	return scanClaim(r.db.QueryRowContext(ctx, q, taskID.String(), claimerID.String()))
}

func (r *claimRepository) CountByTaskID(ctx context.Context, taskID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM claims WHERE task_id = $1::uuid AND status != 'cancelled'`,
		taskID.String(),
	).Scan(&count)
	return count, err
}

func (r *claimRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ClaimStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE claims SET status = $1 WHERE id = $2::uuid`, status, id.String())
	return err
}

func (r *claimRepository) SubmitCompletion(ctx context.Context, id uuid.UUID, text, imageURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE claims SET completion_text = $1, completion_image_url = $2, submitted_at = NOW() WHERE id = $3::uuid`,
		text, imageURL, id.String(),
	)
	return err
}

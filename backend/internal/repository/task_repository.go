package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	GetByOwnerID(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Task, error)
	GetOpenTasks(ctx context.Context, limit, offset int) ([]*domain.Task, error)
	SearchTasks(ctx context.Context, query string, status domain.TaskStatus, limit, offset int) ([]*domain.Task, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TaskStatus) error
	SetEscrowLocked(ctx context.Context, id uuid.UUID, locked bool) error
	GetTasksPastClaimDeadline(ctx context.Context) ([]*domain.Task, error)
	GetTasksPastOwnerDeadline(ctx context.Context) ([]*domain.Task, error)
}

type taskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) TaskRepository {
	return &taskRepository{db: db}
}

func scanTask(row interface {
	Scan(...interface{}) error
}) (*domain.Task, error) {
	task := &domain.Task{}
	err := row.Scan(
		&task.ID, &task.OwnerID, &task.Title, &task.Description,
		&task.RewardAmount, &task.MaxClaimants, &task.ClaimDeadline,
		&task.OwnerDeadline, &task.Status, &task.EscrowLocked,
		&task.CreatedAt, &task.UpdatedAt,
	)
	return task, err
}

func (r *taskRepository) Create(ctx context.Context, task *domain.Task) error {
	query := `
		INSERT INTO tasks (id, owner_id, title, description, reward_amount, max_claimants, claim_deadline, owner_deadline, status, escrow_locked)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		task.ID.String(), task.OwnerID.String(),
		task.Title, task.Description, task.RewardAmount, task.MaxClaimants,
		task.ClaimDeadline, task.OwnerDeadline, task.Status, task.EscrowLocked,
	).Scan(&task.CreatedAt, &task.UpdatedAt)
}

func (r *taskRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	const q = `
		SELECT id, owner_id, title, description, reward_amount, max_claimants, claim_deadline, owner_deadline, status, escrow_locked, created_at, updated_at
		FROM tasks WHERE id = $1::uuid
	`
	task, err := scanTask(r.db.QueryRowContext(ctx, q, id.String()))
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (r *taskRepository) GetByOwnerID(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Task, error) {
	const q = `
		SELECT id, owner_id, title, description, reward_amount, max_claimants, claim_deadline, owner_deadline, status, escrow_locked, created_at, updated_at
		FROM tasks WHERE owner_id = $1::uuid
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, q, ownerID.String(), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (r *taskRepository) GetOpenTasks(ctx context.Context, limit, offset int) ([]*domain.Task, error) {
	const q = `
		SELECT id, owner_id, title, description, reward_amount, max_claimants, claim_deadline, owner_deadline, status, escrow_locked, created_at, updated_at
		FROM tasks WHERE status = 'open' AND claim_deadline > NOW()
		ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (r *taskRepository) SearchTasks(ctx context.Context, query string, status domain.TaskStatus, limit, offset int) ([]*domain.Task, error) {
	// $2 = optional status filter ('' means "any status").
	const q = `
		SELECT id, owner_id, title, description, reward_amount, max_claimants, claim_deadline, owner_deadline, status, escrow_locked, created_at, updated_at
		FROM tasks
		WHERE search_vector @@ plainto_tsquery('english', $1)
		  AND ($2 = '' OR status = $2)
		ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC, created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.db.QueryContext(ctx, q, query, string(status), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (r *taskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TaskStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tasks SET status = $1 WHERE id = $2::uuid`, status, id.String())
	return err
}

func (r *taskRepository) SetEscrowLocked(ctx context.Context, id uuid.UUID, locked bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tasks SET escrow_locked = $1 WHERE id = $2::uuid`, locked, id.String())
	return err
}

func (r *taskRepository) GetTasksPastClaimDeadline(ctx context.Context) ([]*domain.Task, error) {
	const q = `
		SELECT id, owner_id, title, description, reward_amount, max_claimants, claim_deadline, owner_deadline, status, escrow_locked, created_at, updated_at
		FROM tasks WHERE status = 'open' AND claim_deadline <= NOW()
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (r *taskRepository) GetTasksPastOwnerDeadline(ctx context.Context) ([]*domain.Task, error) {
	const q = `
		SELECT id, owner_id, title, description, reward_amount, max_claimants, claim_deadline, owner_deadline, status, escrow_locked, created_at, updated_at
		FROM tasks WHERE status IN ('claimed', 'open') AND owner_deadline <= NOW()
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTasks(rows)
}

func scanTasks(rows *sql.Rows) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for rows.Next() {
		task := &domain.Task{}
		err := rows.Scan(
			&task.ID, &task.OwnerID, &task.Title, &task.Description,
			&task.RewardAmount, &task.MaxClaimants, &task.ClaimDeadline,
			&task.OwnerDeadline, &task.Status, &task.EscrowLocked,
			&task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, rows.Err()
}

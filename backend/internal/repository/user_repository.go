package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
)

type UserRepository interface {
	GetOrCreateByDeviceID(ctx context.Context, deviceID string) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateReputation(ctx context.Context, id uuid.UUID, delta int) error
	UpdateEarnings(ctx context.Context, id uuid.UUID, amount float64) error
	UpdateSpending(ctx context.Context, id uuid.UUID, amount float64) error
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func scanUser(row interface{ Scan(...interface{}) error }) (*domain.User, error) {
	user := &domain.User{}
	err := row.Scan(&user.ID, &user.DeviceID, &user.CreatedAt, &user.Reputation, &user.TotalEarned, &user.TotalSpent)
	return user, err
}

func (r *userRepository) GetOrCreateByDeviceID(ctx context.Context, deviceID string) (*domain.User, error) {
	const q = `
		INSERT INTO users (device_id) VALUES ($1)
		ON CONFLICT (device_id) DO UPDATE SET device_id = users.device_id
		RETURNING id, device_id, created_at, reputation, total_earned, total_spent
	`
	user, err := scanUser(r.db.QueryRowContext(ctx, q, deviceID))
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, device_id, created_at, reputation, total_earned, total_spent
		FROM users WHERE id = $1::uuid
	`
	user, err := scanUser(r.db.QueryRowContext(ctx, q, id.String()))
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepository) UpdateReputation(ctx context.Context, id uuid.UUID, delta int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET reputation = reputation + $1 WHERE id = $2::uuid`, delta, id.String())
	return err
}

func (r *userRepository) UpdateEarnings(ctx context.Context, id uuid.UUID, amount float64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET total_earned = total_earned + $1 WHERE id = $2::uuid`, amount, id.String())
	return err
}

func (r *userRepository) UpdateSpending(ctx context.Context, id uuid.UUID, amount float64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET total_spent = total_spent + $1 WHERE id = $2::uuid`, amount, id.String())
	return err
}

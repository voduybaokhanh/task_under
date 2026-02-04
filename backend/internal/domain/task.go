package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusOpen      TaskStatus = "open"
	TaskStatusClaimed   TaskStatus = "claimed"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusDisputed  TaskStatus = "disputed"
)

type Task struct {
	ID            uuid.UUID  `json:"id"`
	OwnerID       uuid.UUID  `json:"owner_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	RewardAmount  float64    `json:"reward_amount"`
	MaxClaimants  int        `json:"max_claimants"`
	ClaimDeadline time.Time  `json:"claim_deadline"`
	OwnerDeadline time.Time  `json:"owner_deadline"`
	Status        TaskStatus `json:"status"`
	EscrowLocked  bool       `json:"escrow_locked"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (t *Task) CanBeClaimed() bool {
	now := time.Now()
	return t.Status == TaskStatusOpen &&
		now.Before(t.ClaimDeadline) &&
		!t.EscrowLocked
}

func (t *Task) ShouldAutoCancel() bool {
	now := time.Now()
	return t.Status == TaskStatusOpen && now.After(t.ClaimDeadline)
}

package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	DeviceID    string    `json:"device_id"`
	CreatedAt   time.Time `json:"created_at"`
	Reputation  int       `json:"reputation"`
	TotalEarned float64   `json:"total_earned"`
	TotalSpent  float64   `json:"total_spent"`
}

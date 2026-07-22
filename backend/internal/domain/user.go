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
	// PushToken is the device's Expo push token. Never returned to clients.
	PushToken string `json:"-"`
	// PublicKey is the device's X25519 public key (base64) for E2E encrypted
	// chat. Public by nature — other users need it to message this one.
	PublicKey string `json:"public_key"`
}

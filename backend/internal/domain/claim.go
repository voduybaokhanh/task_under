package domain

import (
	"time"

	"github.com/google/uuid"
)

type ClaimStatus string

const (
	ClaimStatusPending  ClaimStatus = "pending"
	ClaimStatusApproved ClaimStatus = "approved"
	ClaimStatusRejected ClaimStatus = "rejected"
	ClaimStatusCancelled ClaimStatus = "cancelled"
)

type Claim struct {
	ID              uuid.UUID  `json:"id"`
	TaskID          uuid.UUID  `json:"task_id"`
	ClaimerID       uuid.UUID  `json:"claimer_id"`
	Status          ClaimStatus `json:"status"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`
	CompletionText  string     `json:"completion_text,omitempty"`
	CompletionImageURL string  `json:"completion_image_url,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (c *Claim) IsSubmitted() bool {
	return c.SubmittedAt != nil && c.CompletionText != ""
}

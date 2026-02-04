package domain

import (
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	ID                    uuid.UUID `json:"id"`
	TaskID                uuid.UUID `json:"task_id"`
	ParticipantID         uuid.UUID `json:"participant_id"`
	OtherParticipantID    uuid.UUID `json:"other_participant_id"`
	DeletedByParticipant  bool      `json:"deleted_by_participant"`
	DeletedByOther        bool      `json:"deleted_by_other"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (c *Chat) IsDeleted() bool {
	return c.DeletedByParticipant || c.DeletedByOther
}

func (c *Chat) IsVisibleTo(userID uuid.UUID) bool {
	if c.IsDeleted() {
		return false
	}
	return c.ParticipantID == userID || c.OtherParticipantID == userID
}

type Message struct {
	ID        uuid.UUID `json:"id"`
	ChatID    uuid.UUID `json:"chat_id"`
	SenderID  uuid.UUID `json:"sender_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

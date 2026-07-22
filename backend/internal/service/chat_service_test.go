package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/task-underground/backend/internal/domain"
)

// chatRepoWithChat serves one fixed chat, so tests control who the
// participants are.
type chatRepoWithChat struct {
	mockChatRepoForClaimSvc
	chat *domain.Chat
}

func (c *chatRepoWithChat) GetByID(context.Context, uuid.UUID) (*domain.Chat, error) {
	return c.chat, nil
}

func TestSendMessageNotifiesTheOtherParticipant(t *testing.T) {
	alice, bob := uuid.New(), uuid.New()
	chat := &domain.Chat{ID: uuid.New(), ParticipantID: alice, OtherParticipantID: bob}
	notifier := &recordingNotifier{}

	svc := NewChatService(&chatRepoWithChat{chat: chat}, notifier)

	// Alice writes: Bob is the one who should hear about it.
	_, err := svc.SendMessage(context.Background(), chat.ID, alice, "hi")
	assert.NoError(t, err)
	assert.Equal(t, []event{{UserID: bob, Type: "chat_message"}}, notifier.events())

	// Bob replies: now Alice is notified, never the sender themselves.
	_, err = svc.SendMessage(context.Background(), chat.ID, bob, "hello")
	assert.NoError(t, err)
	assert.Equal(t, event{UserID: alice, Type: "chat_message"}, notifier.events()[1])
}

func TestSendMessageToChatYouAreNotInIsRejected(t *testing.T) {
	chat := &domain.Chat{ID: uuid.New(), ParticipantID: uuid.New(), OtherParticipantID: uuid.New()}
	notifier := &recordingNotifier{}

	svc := NewChatService(&chatRepoWithChat{chat: chat}, notifier)

	_, err := svc.SendMessage(context.Background(), chat.ID, uuid.New(), "let me in")
	assert.Error(t, err)
	assert.Empty(t, notifier.events(), "a rejected message must not notify anyone")
}

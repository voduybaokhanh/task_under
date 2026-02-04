package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
	"github.com/task-underground/backend/internal/repository"
)

var (
	ErrChatNotFound = errors.New("chat not found")
)

type ChatService interface {
	GetOrCreateChat(ctx context.Context, taskID, userID, otherUserID uuid.UUID) (*domain.Chat, error)
	GetChatsByTaskID(ctx context.Context, taskID, userID uuid.UUID) ([]*domain.Chat, error)
	DeleteChat(ctx context.Context, chatID, userID uuid.UUID) error
	SendMessage(ctx context.Context, chatID, senderID uuid.UUID, content string) (*domain.Message, error)
	GetMessages(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error)
}

type chatService struct {
	chatRepo repository.ChatRepository
}

func NewChatService(chatRepo repository.ChatRepository) ChatService {
	return &chatService{chatRepo: chatRepo}
}

func (s *chatService) GetOrCreateChat(ctx context.Context, taskID, userID, otherUserID uuid.UUID) (*domain.Chat, error) {
	return s.chatRepo.GetOrCreate(ctx, taskID, userID, otherUserID)
}

func (s *chatService) GetChatsByTaskID(ctx context.Context, taskID, userID uuid.UUID) ([]*domain.Chat, error) {
	return s.chatRepo.GetByTaskIDAndUserID(ctx, taskID, userID)
}

func (s *chatService) DeleteChat(ctx context.Context, chatID, userID uuid.UUID) error {
	return s.chatRepo.DeleteForUser(ctx, chatID, userID)
}

func (s *chatService) SendMessage(ctx context.Context, chatID, senderID uuid.UUID, content string) (*domain.Message, error) {
	if content == "" {
		return nil, errors.New("message content cannot be empty")
	}

	// Verify chat exists and user is participant
	chat, err := s.chatRepo.GetByID(ctx, chatID)
	if err != nil {
		return nil, ErrChatNotFound
	}

	if chat.ParticipantID != senderID && chat.OtherParticipantID != senderID {
		return nil, errors.New("unauthorized")
	}

	if !chat.IsVisibleTo(senderID) {
		return nil, errors.New("chat is deleted")
	}

	message := &domain.Message{
		ID:       uuid.New(),
		ChatID:   chatID,
		SenderID: senderID,
		Content:  content,
	}

	err = s.chatRepo.CreateMessage(ctx, message)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (s *chatService) GetMessages(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.chatRepo.GetMessagesByChatID(ctx, chatID, limit, offset)
}

package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
)

type ChatRepository interface {
	GetOrCreate(ctx context.Context, taskID, participantID, otherParticipantID uuid.UUID) (*domain.Chat, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error)
	GetByTaskIDAndUserID(ctx context.Context, taskID, userID uuid.UUID) ([]*domain.Chat, error)
	DeleteForUser(ctx context.Context, chatID, userID uuid.UUID) error
	CreateMessage(ctx context.Context, message *domain.Message) error
	GetMessagesByChatID(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error)
}

type chatRepository struct {
	db *sql.DB
}

func NewChatRepository(db *sql.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) GetOrCreate(ctx context.Context, taskID, participantID, otherParticipantID uuid.UUID) (*domain.Chat, error) {
	// Try to find existing chat
	query := `
		SELECT id, task_id, participant_id, other_participant_id, deleted_by_participant, deleted_by_other, created_at, updated_at
		FROM chats
		WHERE task_id = $1 AND participant_id = $2 AND other_participant_id = $3
	`
	
	chat := &domain.Chat{}
	err := r.db.QueryRowContext(ctx, query, taskID, participantID, otherParticipantID).Scan(
		&chat.ID,
		&chat.TaskID,
		&chat.ParticipantID,
		&chat.OtherParticipantID,
		&chat.DeletedByParticipant,
		&chat.DeletedByOther,
		&chat.CreatedAt,
		&chat.UpdatedAt,
	)
	
	if err == nil {
		// Reset deletion flags if chat exists
		if chat.DeletedByParticipant || chat.DeletedByOther {
			updateQuery := `
				UPDATE chats 
				SET deleted_by_participant = FALSE, deleted_by_other = FALSE
				WHERE id = $1
			`
			r.db.ExecContext(ctx, updateQuery, chat.ID)
			chat.DeletedByParticipant = false
			chat.DeletedByOther = false
		}
		return chat, nil
	}
	
	if err != sql.ErrNoRows {
		return nil, err
	}
	
	// Create new chat
	chat.ID = uuid.New()
	chat.TaskID = taskID
	chat.ParticipantID = participantID
	chat.OtherParticipantID = otherParticipantID
	
	insertQuery := `
		INSERT INTO chats (id, task_id, participant_id, other_participant_id)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`
	
	err = r.db.QueryRowContext(ctx, insertQuery,
		chat.ID,
		chat.TaskID,
		chat.ParticipantID,
		chat.OtherParticipantID,
	).Scan(&chat.CreatedAt, &chat.UpdatedAt)
	
	return chat, err
}

func (r *chatRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {
	query := `
		SELECT id, task_id, participant_id, other_participant_id, deleted_by_participant, deleted_by_other, created_at, updated_at
		FROM chats
		WHERE id = $1
	`
	
	chat := &domain.Chat{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&chat.ID,
		&chat.TaskID,
		&chat.ParticipantID,
		&chat.OtherParticipantID,
		&chat.DeletedByParticipant,
		&chat.DeletedByOther,
		&chat.CreatedAt,
		&chat.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (r *chatRepository) GetByTaskIDAndUserID(ctx context.Context, taskID, userID uuid.UUID) ([]*domain.Chat, error) {
	query := `
		SELECT id, task_id, participant_id, other_participant_id, deleted_by_participant, deleted_by_other, created_at, updated_at
		FROM chats
		WHERE task_id = $1 AND (participant_id = $2 OR other_participant_id = $2)
		AND NOT (deleted_by_participant = TRUE AND participant_id = $2)
		AND NOT (deleted_by_other = TRUE AND other_participant_id = $2)
		ORDER BY updated_at DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query, taskID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var chats []*domain.Chat
	for rows.Next() {
		chat := &domain.Chat{}
		err := rows.Scan(
			&chat.ID,
			&chat.TaskID,
			&chat.ParticipantID,
			&chat.OtherParticipantID,
			&chat.DeletedByParticipant,
			&chat.DeletedByOther,
			&chat.CreatedAt,
			&chat.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (r *chatRepository) DeleteForUser(ctx context.Context, chatID, userID uuid.UUID) error {
	query := `
		UPDATE chats
		SET deleted_by_participant = CASE WHEN participant_id = $2 THEN TRUE ELSE deleted_by_participant END,
		    deleted_by_other = CASE WHEN other_participant_id = $2 THEN TRUE ELSE deleted_by_other END
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, chatID, userID)
	return err
}

func (r *chatRepository) CreateMessage(ctx context.Context, message *domain.Message) error {
	query := `
		INSERT INTO messages (id, chat_id, sender_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`
	
	err := r.db.QueryRowContext(ctx, query,
		message.ID,
		message.ChatID,
		message.SenderID,
		message.Content,
	).Scan(&message.CreatedAt)
	
	return err
}

func (r *chatRepository) GetMessagesByChatID(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	query := `
		SELECT id, chat_id, sender_id, content, created_at
		FROM messages
		WHERE chat_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.db.QueryContext(ctx, query, chatID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var messages []*domain.Message
	for rows.Next() {
		message := &domain.Message{}
		err := rows.Scan(
			&message.ID,
			&message.ChatID,
			&message.SenderID,
			&message.Content,
			&message.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

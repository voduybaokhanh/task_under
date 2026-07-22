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

func scanChat(row interface{ Scan(...interface{}) error }) (*domain.Chat, error) {
	chat := &domain.Chat{}
	err := row.Scan(
		&chat.ID, &chat.TaskID, &chat.ParticipantID, &chat.OtherParticipantID,
		&chat.DeletedByParticipant, &chat.DeletedByOther, &chat.CreatedAt, &chat.UpdatedAt,
	)
	return chat, err
}

func (r *chatRepository) GetOrCreate(ctx context.Context, taskID, participantID, otherParticipantID uuid.UUID) (*domain.Chat, error) {
	const selectQ = `
		SELECT id, task_id, participant_id, other_participant_id, deleted_by_participant, deleted_by_other, created_at, updated_at
		FROM chats WHERE task_id = $1::uuid AND participant_id = $2::uuid AND other_participant_id = $3::uuid
	`
	chat, err := scanChat(r.db.QueryRowContext(ctx, selectQ, taskID.String(), participantID.String(), otherParticipantID.String()))
	if err == nil {
		if chat.DeletedByParticipant || chat.DeletedByOther {
			r.db.ExecContext(ctx,
				`UPDATE chats SET deleted_by_participant = FALSE, deleted_by_other = FALSE WHERE id = $1::uuid`,
				chat.ID.String(),
			)
			chat.DeletedByParticipant = false
			chat.DeletedByOther = false
		}
		return chat, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	chat = &domain.Chat{
		ID:                 uuid.New(),
		TaskID:             taskID,
		ParticipantID:      participantID,
		OtherParticipantID: otherParticipantID,
	}
	const insertQ = `
		INSERT INTO chats (id, task_id, participant_id, other_participant_id)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid)
		RETURNING created_at, updated_at
	`
	err = r.db.QueryRowContext(ctx, insertQ,
		chat.ID.String(), chat.TaskID.String(), chat.ParticipantID.String(), chat.OtherParticipantID.String(),
	).Scan(&chat.CreatedAt, &chat.UpdatedAt)
	return chat, err
}

func (r *chatRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Chat, error) {
	const q = `
		SELECT id, task_id, participant_id, other_participant_id, deleted_by_participant, deleted_by_other, created_at, updated_at
		FROM chats WHERE id = $1::uuid
	`
	chat, err := scanChat(r.db.QueryRowContext(ctx, q, id.String()))
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (r *chatRepository) GetByTaskIDAndUserID(ctx context.Context, taskID, userID uuid.UUID) ([]*domain.Chat, error) {
	const q = `
		SELECT id, task_id, participant_id, other_participant_id, deleted_by_participant, deleted_by_other, created_at, updated_at
		FROM chats
		WHERE task_id = $1::uuid AND (participant_id = $2::uuid OR other_participant_id = $2::uuid)
		AND NOT (deleted_by_participant = TRUE AND participant_id = $2::uuid)
		AND NOT (deleted_by_other = TRUE AND other_participant_id = $2::uuid)
		ORDER BY updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, q, taskID.String(), userID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []*domain.Chat
	for rows.Next() {
		chat, err := scanChat(rows)
		if err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (r *chatRepository) DeleteForUser(ctx context.Context, chatID, userID uuid.UUID) error {
	const q = `
		UPDATE chats
		SET deleted_by_participant = CASE WHEN participant_id = $2::uuid THEN TRUE ELSE deleted_by_participant END,
		    deleted_by_other = CASE WHEN other_participant_id = $2::uuid THEN TRUE ELSE deleted_by_other END
		WHERE id = $1::uuid
	`
	_, err := r.db.ExecContext(ctx, q, chatID.String(), userID.String())
	return err
}

func (r *chatRepository) CreateMessage(ctx context.Context, message *domain.Message) error {
	const q = `
		INSERT INTO messages (id, chat_id, sender_id, content)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4)
		RETURNING created_at
	`
	return r.db.QueryRowContext(ctx, q,
		message.ID.String(), message.ChatID.String(), message.SenderID.String(), message.Content,
	).Scan(&message.CreatedAt)
}

func (r *chatRepository) GetMessagesByChatID(ctx context.Context, chatID uuid.UUID, limit, offset int) ([]*domain.Message, error) {
	const q = `
		SELECT id, chat_id, sender_id, content, created_at
		FROM messages WHERE chat_id = $1::uuid
		ORDER BY created_at ASC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, q, chatID.String(), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		msg := &domain.Message{}
		err := rows.Scan(&msg.ID, &msg.ChatID, &msg.SenderID, &msg.Content, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

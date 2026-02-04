package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/task-underground/backend/internal/domain"
)

type mockClaimRepoForClaimSvc struct {
	claims map[uuid.UUID]*domain.Claim
}

func (m *mockClaimRepoForClaimSvc) Create(ctx context.Context, claim *domain.Claim) error {
	m.claims[claim.ID] = claim
	return nil
}

func (m *mockClaimRepoForClaimSvc) GetByID(ctx context.Context, id uuid.UUID) (*domain.Claim, error) {
	claim, ok := m.claims[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return claim, nil
}

func (m *mockClaimRepoForClaimSvc) GetByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.Claim, error) {
	var result []*domain.Claim
	for _, claim := range m.claims {
		if claim.TaskID == taskID {
			result = append(result, claim)
		}
	}
	return result, nil
}

func (m *mockClaimRepoForClaimSvc) GetByTaskIDAndClaimerID(ctx context.Context, taskID, claimerID uuid.UUID) (*domain.Claim, error) {
	for _, claim := range m.claims {
		if claim.TaskID == taskID && claim.ClaimerID == claimerID {
			return claim, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockClaimRepoForClaimSvc) CountByTaskID(ctx context.Context, taskID uuid.UUID) (int, error) {
	count := 0
	for _, claim := range m.claims {
		if claim.TaskID == taskID && claim.Status != domain.ClaimStatusCancelled {
			count++
		}
	}
	return count, nil
}

func (m *mockClaimRepoForClaimSvc) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ClaimStatus) error {
	claim, ok := m.claims[id]
	if !ok {
		return sql.ErrNoRows
	}
	claim.Status = status
	return nil
}

func (m *mockClaimRepoForClaimSvc) SubmitCompletion(ctx context.Context, id uuid.UUID, text, imageURL string) error {
	claim, ok := m.claims[id]
	if !ok {
		return sql.ErrNoRows
	}
	now := time.Now()
	claim.CompletionText = text
	claim.CompletionImageURL = imageURL
	claim.SubmittedAt = &now
	return nil
}

type mockTaskRepoForClaimSvc struct {
	tasks map[uuid.UUID]*domain.Task
}

func (m *mockTaskRepoForClaimSvc) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	task, ok := m.tasks[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return task, nil
}

func (m *mockTaskRepoForClaimSvc) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TaskStatus) error {
	task, ok := m.tasks[id]
	if !ok {
		return sql.ErrNoRows
	}
	task.Status = status
	return nil
}

type mockChatRepoForClaimSvc struct{}

func (m *mockChatRepoForClaimSvc) GetOrCreate(ctx context.Context, taskID, participantID, otherParticipantID uuid.UUID) (*domain.Chat, error) {
	return &domain.Chat{
		ID:                 uuid.New(),
		TaskID:             taskID,
		ParticipantID:      participantID,
		OtherParticipantID: otherParticipantID,
	}, nil
}

func TestClaimTask(t *testing.T) {
	claimRepo := &mockClaimRepoForClaimSvc{claims: make(map[uuid.UUID]*domain.Claim)}
	taskRepo := &mockTaskRepoForClaimSvc{tasks: make(map[uuid.UUID]*domain.Task)}
	chatRepo := &mockChatRepoForClaimSvc{}
	escrowSvc := &mockEscrowSvc{}
	userRepo := &mockUserRepo{}

	service := NewClaimService(claimRepo, taskRepo, chatRepo, escrowSvc, userRepo)

	ownerID := uuid.New()
	claimerID := uuid.New()
	taskID := uuid.New()

	task := &domain.Task{
		ID:            taskID,
		OwnerID:       ownerID,
		Title:         "Test Task",
		Description:   "Test",
		RewardAmount:  100.0,
		MaxClaimants:  1,
		ClaimDeadline: time.Now().Add(7 * 24 * time.Hour),
		OwnerDeadline: time.Now().Add(30 * 24 * time.Hour),
		Status:        domain.TaskStatusOpen,
		EscrowLocked:  true,
	}
	taskRepo.tasks[taskID] = task

	claim, err := service.ClaimTask(context.Background(), taskID, claimerID)
	assert.NoError(t, err)
	assert.NotNil(t, claim)
	assert.Equal(t, taskID, claim.TaskID)
	assert.Equal(t, claimerID, claim.ClaimerID)
	assert.Equal(t, domain.ClaimStatusPending, claim.Status)

	// Task status should be updated to claimed
	updatedTask, _ := taskRepo.GetByID(context.Background(), taskID)
	assert.Equal(t, domain.TaskStatusClaimed, updatedTask.Status)
}

func TestClaimTaskLimitReached(t *testing.T) {
	claimRepo := &mockClaimRepoForClaimSvc{claims: make(map[uuid.UUID]*domain.Claim)}
	taskRepo := &mockTaskRepoForClaimSvc{tasks: make(map[uuid.UUID]*domain.Task)}
	chatRepo := &mockChatRepoForClaimSvc{}
	escrowSvc := &mockEscrowSvc{}
	userRepo := &mockUserRepo{}

	service := NewClaimService(claimRepo, taskRepo, chatRepo, escrowSvc, userRepo)

	ownerID := uuid.New()
	claimerID1 := uuid.New()
	claimerID2 := uuid.New()
	taskID := uuid.New()

	task := &domain.Task{
		ID:            taskID,
		OwnerID:       ownerID,
		Title:         "Test Task",
		Description:   "Test",
		RewardAmount:  100.0,
		MaxClaimants:  1,
		ClaimDeadline: time.Now().Add(7 * 24 * time.Hour),
		OwnerDeadline: time.Now().Add(30 * 24 * time.Hour),
		Status:        domain.TaskStatusOpen,
		EscrowLocked:  true,
	}
	taskRepo.tasks[taskID] = task

	// First claim succeeds
	_, err := service.ClaimTask(context.Background(), taskID, claimerID1)
	assert.NoError(t, err)

	// Second claim should fail
	_, err = service.ClaimTask(context.Background(), taskID, claimerID2)
	assert.Error(t, err)
	assert.Equal(t, ErrClaimLimitReached, err)
}

type mockUserRepo struct{}

func (m *mockUserRepo) GetOrCreateByDeviceID(ctx context.Context, deviceID string) (*domain.User, error) {
	return &domain.User{ID: uuid.New()}, nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return &domain.User{ID: id}, nil
}

func (m *mockUserRepo) UpdateReputation(ctx context.Context, id uuid.UUID, delta int) error {
	return nil
}

func (m *mockUserRepo) UpdateEarnings(ctx context.Context, id uuid.UUID, amount float64) error {
	return nil
}

func (m *mockUserRepo) UpdateSpending(ctx context.Context, id uuid.UUID, amount float64) error {
	return nil
}

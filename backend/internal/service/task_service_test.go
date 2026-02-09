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

// Mock repositories for testing
type mockTaskRepo struct {
	tasks map[uuid.UUID]*domain.Task
}

func (m *mockTaskRepo) Create(ctx context.Context, task *domain.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	task, ok := m.tasks[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return task, nil
}

func (m *mockTaskRepo) GetByOwnerID(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Task, error) {
	var result []*domain.Task
	for _, task := range m.tasks {
		if task.OwnerID == ownerID {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockTaskRepo) GetOpenTasks(ctx context.Context, limit, offset int) ([]*domain.Task, error) {
	var result []*domain.Task
	for _, task := range m.tasks {
		if task.Status == domain.TaskStatusOpen {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockTaskRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TaskStatus) error {
	task, ok := m.tasks[id]
	if !ok {
		return sql.ErrNoRows
	}
	task.Status = status
	return nil
}

func (m *mockTaskRepo) SetEscrowLocked(ctx context.Context, id uuid.UUID, locked bool) error {
	task, ok := m.tasks[id]
	if !ok {
		return sql.ErrNoRows
	}
	task.EscrowLocked = locked
	return nil
}

func (m *mockTaskRepo) GetTasksPastClaimDeadline(ctx context.Context) ([]*domain.Task, error) {
	var result []*domain.Task
	now := time.Now()
	for _, task := range m.tasks {
		if task.Status == domain.TaskStatusOpen && task.ClaimDeadline.Before(now) {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *mockTaskRepo) GetTasksPastOwnerDeadline(ctx context.Context) ([]*domain.Task, error) {
	var result []*domain.Task
	now := time.Now()
	for _, task := range m.tasks {
		if (task.Status == domain.TaskStatusClaimed || task.Status == domain.TaskStatusOpen) && task.OwnerDeadline.Before(now) {
			result = append(result, task)
		}
	}
	return result, nil
}

type mockClaimRepo struct {
	claims map[uuid.UUID]*domain.Claim
}

func (m *mockClaimRepo) Create(ctx context.Context, claim *domain.Claim) error {
	m.claims[claim.ID] = claim
	return nil
}

func (m *mockClaimRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Claim, error) {
	claim, ok := m.claims[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return claim, nil
}

func (m *mockClaimRepo) GetByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.Claim, error) {
	var result []*domain.Claim
	for _, claim := range m.claims {
		if claim.TaskID == taskID {
			result = append(result, claim)
		}
	}
	return result, nil
}

func (m *mockClaimRepo) GetByTaskIDAndClaimerID(ctx context.Context, taskID, claimerID uuid.UUID) (*domain.Claim, error) {
	for _, claim := range m.claims {
		if claim.TaskID == taskID && claim.ClaimerID == claimerID {
			return claim, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (m *mockClaimRepo) CountByTaskID(ctx context.Context, taskID uuid.UUID) (int, error) {
	count := 0
	for _, claim := range m.claims {
		if claim.TaskID == taskID && claim.Status != domain.ClaimStatusCancelled {
			count++
		}
	}
	return count, nil
}

func (m *mockClaimRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ClaimStatus) error {
	claim, ok := m.claims[id]
	if !ok {
		return sql.ErrNoRows
	}
	claim.Status = status
	return nil
}

func (m *mockClaimRepo) SubmitCompletion(ctx context.Context, id uuid.UUID, text, imageURL string) error {
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

type mockEscrowSvc struct{}

func (m *mockEscrowSvc) LockEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error {
	return nil
}

func (m *mockEscrowSvc) ReleaseEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error {
	return nil
}

func (m *mockEscrowSvc) RefundEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error {
	return nil
}

func TestAutoCancelExpiredTasks(t *testing.T) {
	taskRepo := &mockTaskRepo{tasks: make(map[uuid.UUID]*domain.Task)}
	claimRepo := &mockClaimRepo{claims: make(map[uuid.UUID]*domain.Claim)}
	escrowSvc := &mockEscrowSvc{}

	service := NewTaskService(taskRepo, claimRepo, escrowSvc)

	ownerID := uuid.New()
	pastDeadline := time.Now().Add(-1 * time.Hour)
	futureDeadline := time.Now().Add(1 * time.Hour)

	// Create task with expired claim deadline and no claims
	taskID := uuid.New()
	task := &domain.Task{
		ID:            taskID,
		OwnerID:       ownerID,
		Title:         "Test Task",
		Description:   "Test",
		RewardAmount:  100.0,
		MaxClaimants:  1,
		ClaimDeadline: pastDeadline,
		OwnerDeadline: futureDeadline,
		Status:        domain.TaskStatusOpen,
		EscrowLocked:  true,
	}
	taskRepo.tasks[taskID] = task

	err := service.AutoCancelExpiredTasks(context.Background())
	assert.NoError(t, err)

	// Task should be cancelled
	updatedTask, _ := taskRepo.GetByID(context.Background(), taskID)
	assert.Equal(t, domain.TaskStatusCancelled, updatedTask.Status)
}

func TestCreateTask(t *testing.T) {
	taskRepo := &mockTaskRepo{tasks: make(map[uuid.UUID]*domain.Task)}
	claimRepo := &mockClaimRepo{claims: make(map[uuid.UUID]*domain.Claim)}
	escrowSvc := &mockEscrowSvc{}

	service := NewTaskService(taskRepo, claimRepo, escrowSvc)

	ownerID := uuid.New()
	req := CreateTaskRequest{
		Title:         "Test Task",
		Description:   "Test Description",
		RewardAmount:  100.0,
		MaxClaimants:  1,
		ClaimDeadline: time.Now().Add(7 * 24 * time.Hour),
		OwnerDeadline: time.Now().Add(30 * 24 * time.Hour),
	}

	task, err := service.CreateTask(context.Background(), ownerID, req)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, req.Title, task.Title)
	assert.Equal(t, domain.TaskStatusOpen, task.Status)
}

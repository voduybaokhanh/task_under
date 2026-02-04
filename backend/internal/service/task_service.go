package service

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
	"github.com/task-underground/backend/internal/repository"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrInvalidTask       = errors.New("invalid task")
	ErrTaskNotClaimable  = errors.New("task cannot be claimed")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrTaskAlreadyLocked = errors.New("task escrow already locked")
)

type TaskService interface {
	CreateTask(ctx context.Context, ownerID uuid.UUID, req CreateTaskRequest) (*domain.Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	GetOpenTasks(ctx context.Context, limit, offset int) ([]*domain.Task, error)
	GetUserTasks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Task, error)
	AutoCancelExpiredTasks(ctx context.Context) error
}

type CreateTaskRequest struct {
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	RewardAmount  float64   `json:"reward_amount"`
	MaxClaimants  int       `json:"max_claimants"`
	ClaimDeadline time.Time `json:"claim_deadline"`
	OwnerDeadline time.Time `json:"owner_deadline"`
}

type taskService struct {
	taskRepo  repository.TaskRepository
	claimRepo repository.ClaimRepository
	escrowSvc EscrowService
}

func NewTaskService(
	taskRepo repository.TaskRepository,
	claimRepo repository.ClaimRepository,
	escrowSvc EscrowService,
) TaskService {
	return &taskService{
		taskRepo:  taskRepo,
		claimRepo: claimRepo,
		escrowSvc: escrowSvc,
	}
}

func (s *taskService) CreateTask(ctx context.Context, ownerID uuid.UUID, req CreateTaskRequest) (*domain.Task, error) {
	// Validate
	if req.Title == "" || len(req.Title) > 500 {
		return nil, errors.New("title must be between 1 and 500 characters")
	}
	if req.Description == "" {
		return nil, errors.New("description is required")
	}
	if req.RewardAmount <= 0 {
		return nil, errors.New("reward_amount must be positive")
	}
	if req.MaxClaimants <= 0 {
		return nil, errors.New("max_claimants must be positive")
	}
	now := time.Now()
	if req.ClaimDeadline.Before(now) {
		return nil, errors.New("claim_deadline must be in the future")
	}
	if req.OwnerDeadline.Before(req.ClaimDeadline) {
		return nil, errors.New("owner_deadline must be after claim_deadline")
	}

	task := &domain.Task{
		ID:            uuid.New(),
		OwnerID:       ownerID,
		Title:         req.Title,
		Description:   req.Description,
		RewardAmount:  req.RewardAmount,
		MaxClaimants:  req.MaxClaimants,
		ClaimDeadline: req.ClaimDeadline,
		OwnerDeadline: req.OwnerDeadline,
		Status:        domain.TaskStatusOpen,
		EscrowLocked:  false,
	}

	err := s.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, err
	}

	// Lock escrow
	err = s.escrowSvc.LockEscrow(ctx, task.ID, ownerID, req.RewardAmount)
	if err != nil {
		// Rollback task creation if escrow fails
		s.taskRepo.UpdateStatus(ctx, task.ID, domain.TaskStatusCancelled)
		return nil, err
	}

	return task, nil
}

func (s *taskService) GetTask(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task, nil
}

func (s *taskService) GetOpenTasks(ctx context.Context, limit, offset int) ([]*domain.Task, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.taskRepo.GetOpenTasks(ctx, limit, offset)
}

func (s *taskService) GetUserTasks(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Task, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.taskRepo.GetByOwnerID(ctx, userID, limit, offset)
}

func (s *taskService) AutoCancelExpiredTasks(ctx context.Context) error {
	tasks, err := s.taskRepo.GetTasksPastClaimDeadline(ctx)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		claimCount, err := s.claimRepo.CountByTaskID(ctx, task.ID)
		if err != nil {
			continue
		}

		if claimCount == 0 {
			// No claims, auto-cancel and refund
			err = s.taskRepo.UpdateStatus(ctx, task.ID, domain.TaskStatusCancelled)
			if err != nil {
				continue
			}
			s.escrowSvc.RefundEscrow(ctx, task.ID, task.OwnerID, task.RewardAmount)
		}
	}

	return nil
}

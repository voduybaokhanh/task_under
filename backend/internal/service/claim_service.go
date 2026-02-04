package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
	"github.com/task-underground/backend/internal/repository"
)

var (
	ErrClaimNotFound     = errors.New("claim not found")
	ErrAlreadyClaimed   = errors.New("task already claimed by this user")
	ErrClaimLimitReached = errors.New("claim limit reached")
	ErrInvalidCompletion = errors.New("invalid completion submission")
)

type ClaimService interface {
	ClaimTask(ctx context.Context, taskID, claimerID uuid.UUID) (*domain.Claim, error)
	GetClaim(ctx context.Context, id uuid.UUID) (*domain.Claim, error)
	GetClaimsByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.Claim, error)
	SubmitCompletion(ctx context.Context, claimID, userID uuid.UUID, text, imageURL string) (*domain.Claim, error)
	ApproveClaim(ctx context.Context, claimID, ownerID uuid.UUID) error
	RejectClaim(ctx context.Context, claimID, ownerID uuid.UUID) error
}

type claimService struct {
	claimRepo repository.ClaimRepository
	taskRepo  repository.TaskRepository
	chatRepo  repository.ChatRepository
	escrowSvc EscrowService
	userRepo  repository.UserRepository
}

func NewClaimService(
	claimRepo repository.ClaimRepository,
	taskRepo repository.TaskRepository,
	chatRepo repository.ChatRepository,
	escrowSvc EscrowService,
	userRepo repository.UserRepository,
) ClaimService {
	return &claimService{
		claimRepo: claimRepo,
		taskRepo:  taskRepo,
		chatRepo:  chatRepo,
		escrowSvc: escrowSvc,
		userRepo:  userRepo,
	}
}

func (s *claimService) ClaimTask(ctx context.Context, taskID, claimerID uuid.UUID) (*domain.Claim, error) {
	// Get task
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	// Check if task can be claimed
	if !task.CanBeClaimed() {
		return nil, ErrTaskNotClaimable
	}

	// Check if user already claimed
	existing, err := s.claimRepo.GetByTaskIDAndClaimerID(ctx, taskID, claimerID)
	if err == nil {
		return existing, ErrAlreadyClaimed
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Check claim count
	count, err := s.claimRepo.CountByTaskID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if count >= task.MaxClaimants {
		return nil, ErrClaimLimitReached
	}

	// Create claim
	claim := &domain.Claim{
		ID:        uuid.New(),
		TaskID:    taskID,
		ClaimerID: claimerID,
		Status:    domain.ClaimStatusPending,
	}

	err = s.claimRepo.Create(ctx, claim)
	if err != nil {
		return nil, err
	}

	// Update task status if first claim
	if count == 0 {
		err = s.taskRepo.UpdateStatus(ctx, taskID, domain.TaskStatusClaimed)
		if err != nil {
			return nil, err
		}
	}

	return claim, nil
}

func (s *claimService) GetClaim(ctx context.Context, id uuid.UUID) (*domain.Claim, error) {
	claim, err := s.claimRepo.GetByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrClaimNotFound
		}
		return nil, err
	}
	return claim, nil
}

func (s *claimService) GetClaimsByTaskID(ctx context.Context, taskID uuid.UUID) ([]*domain.Claim, error) {
	return s.claimRepo.GetByTaskID(ctx, taskID)
}

func (s *claimService) SubmitCompletion(ctx context.Context, claimID, userID uuid.UUID, text, imageURL string) (*domain.Claim, error) {
	claim, err := s.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrClaimNotFound
		}
		return nil, err
	}

	if claim.ClaimerID != userID {
		return nil, ErrUnauthorized
	}

	if text == "" {
		return nil, errors.New("completion text is required")
	}

	err = s.claimRepo.SubmitCompletion(ctx, claimID, text, imageURL)
	if err != nil {
		return nil, err
	}

	// Get task to find owner
	task, err := s.taskRepo.GetByID(ctx, claim.TaskID)
	if err != nil {
		return nil, err
	}

	// Open/reopen chat (chat will be created/retrieved)
	_, err = s.chatRepo.GetOrCreate(ctx, claim.TaskID, claim.ClaimerID, task.OwnerID)
	if err != nil {
		// Log error but don't fail completion submission
		// Chat can be created later
	}

	// Refresh claim
	return s.claimRepo.GetByID(ctx, claimID)
}

func (s *claimService) ApproveClaim(ctx context.Context, claimID, ownerID uuid.UUID) error {
	claim, err := s.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrClaimNotFound
		}
		return err
	}

	task, err := s.taskRepo.GetByID(ctx, claim.TaskID)
	if err != nil {
		return err
	}

	if task.OwnerID != ownerID {
		return ErrUnauthorized
	}

	if !claim.IsSubmitted() {
		return errors.New("claim has not been submitted")
	}

	// Update claim status
	err = s.claimRepo.UpdateStatus(ctx, claimID, domain.ClaimStatusApproved)
	if err != nil {
		return err
	}

	// Release escrow to claimer
	err = s.escrowSvc.ReleaseEscrow(ctx, task.ID, claim.ClaimerID, task.RewardAmount)
	if err != nil {
		return err
	}

	// Update user stats
	err = s.userRepo.UpdateEarnings(ctx, claim.ClaimerID, task.RewardAmount)
	if err != nil {
		return err
	}
	err = s.userRepo.UpdateReputation(ctx, claim.ClaimerID, 1)
	if err != nil {
		return err
	}

	// Update task status
	err = s.taskRepo.UpdateStatus(ctx, task.ID, domain.TaskStatusCompleted)
	if err != nil {
		return err
	}

	return nil
}

func (s *claimService) RejectClaim(ctx context.Context, claimID, ownerID uuid.UUID) error {
	claim, err := s.claimRepo.GetByID(ctx, claimID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrClaimNotFound
		}
		return err
	}

	task, err := s.taskRepo.GetByID(ctx, claim.TaskID)
	if err != nil {
		return err
	}

	if task.OwnerID != ownerID {
		return ErrUnauthorized
	}

	err = s.claimRepo.UpdateStatus(ctx, claimID, domain.ClaimStatusRejected)
	if err != nil {
		return err
	}

	return nil
}

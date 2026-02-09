package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/task-underground/backend/internal/domain"
	"github.com/task-underground/backend/internal/repository"
)

var (
	ErrEscrowAlreadyLocked = errors.New("escrow already locked")
	ErrEscrowNotLocked     = errors.New("escrow not locked")
)

type EscrowService interface {
	LockEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error
	ReleaseEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error
	RefundEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error
}

type escrowService struct {
	escrowRepo repository.EscrowRepository
	taskRepo   repository.TaskRepository
}

func NewEscrowService(
	escrowRepo repository.EscrowRepository,
	taskRepo repository.TaskRepository,
) EscrowService {
	return &escrowService{
		escrowRepo: escrowRepo,
		taskRepo:   taskRepo,
	}
}

func (s *escrowService) LockEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error {
	// Check if already locked
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task.EscrowLocked {
		return ErrEscrowAlreadyLocked
	}

	// Create lock transaction
	tx := &domain.EscrowTransaction{
		ID:              uuid.New(),
		TaskID:          taskID,
		UserID:          userID,
		Amount:          amount,
		TransactionType: domain.EscrowTypeLock,
		Status:          domain.EscrowStatusPending,
	}

	err = s.escrowRepo.CreateTransaction(ctx, tx)
	if err != nil {
		return err
	}

	// Mark as locked
	err = s.taskRepo.SetEscrowLocked(ctx, taskID, true)
	if err != nil {
		return err
	}

	// Mark transaction as completed
	err = s.escrowRepo.UpdateTransactionStatus(ctx, tx.ID, domain.EscrowStatusCompleted)
	if err != nil {
		return err
	}

	return nil
}

func (s *escrowService) ReleaseEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error {
	_, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	tx := &domain.EscrowTransaction{
		ID:              uuid.New(),
		TaskID:          taskID,
		UserID:          userID,
		Amount:          amount,
		TransactionType: domain.EscrowTypeRelease,
		Status:          domain.EscrowStatusPending,
	}

	err = s.escrowRepo.CreateTransaction(ctx, tx)
	if err != nil {
		return err
	}

	err = s.escrowRepo.UpdateTransactionStatus(ctx, tx.ID, domain.EscrowStatusCompleted)
	if err != nil {
		return err
	}

	return nil
}

func (s *escrowService) RefundEscrow(ctx context.Context, taskID, userID uuid.UUID, amount float64) error {
	tx := &domain.EscrowTransaction{
		ID:              uuid.New(),
		TaskID:          taskID,
		UserID:          userID,
		Amount:          amount,
		TransactionType: domain.EscrowTypeRefund,
		Status:          domain.EscrowStatusPending,
	}

	err := s.escrowRepo.CreateTransaction(ctx, tx)
	if err != nil {
		return err
	}

	err = s.escrowRepo.UpdateTransactionStatus(ctx, tx.ID, domain.EscrowStatusCompleted)
	if err != nil {
		return err
	}

	// Unlock escrow
	err = s.taskRepo.SetEscrowLocked(ctx, taskID, false)
	if err != nil {
		return err
	}

	return nil
}

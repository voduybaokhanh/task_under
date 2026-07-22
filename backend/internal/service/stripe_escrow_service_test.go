package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v82"
	"github.com/task-underground/backend/internal/domain"
)

// fakeStripe records what the service asked Stripe to do.
type fakeStripe struct {
	created     []*stripe.PaymentIntentParams
	captured    map[string]int64
	cancelled   []string
	refunded    map[string]int64
	transfers   []*stripe.TransferParams
	accounts    []*stripe.AccountParams
	links       []*stripe.AccountLinkParams
	payoutsOn   bool
	failWith    error
	transferErr error
}

func newFakeStripe() *fakeStripe {
	return &fakeStripe{captured: map[string]int64{}, refunded: map[string]int64{}}
}

func (f *fakeStripe) CreatePaymentIntent(_ context.Context, params *stripe.PaymentIntentParams) (*stripe.PaymentIntent, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.created = append(f.created, params)
	return &stripe.PaymentIntent{ID: "pi_test_123"}, nil
}

func (f *fakeStripe) CapturePaymentIntent(_ context.Context, id string, params *stripe.PaymentIntentCaptureParams) (*stripe.PaymentIntent, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.captured[id] = *params.AmountToCapture
	return &stripe.PaymentIntent{ID: id, LatestCharge: &stripe.Charge{ID: "ch_test_1"}}, nil
}

func (f *fakeStripe) CancelPaymentIntent(_ context.Context, id string, _ *stripe.PaymentIntentCancelParams) (*stripe.PaymentIntent, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.cancelled = append(f.cancelled, id)
	return &stripe.PaymentIntent{ID: id}, nil
}

func (f *fakeStripe) CreateRefund(_ context.Context, params *stripe.RefundParams) (*stripe.Refund, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.refunded[*params.PaymentIntent] = *params.Amount
	return &stripe.Refund{ID: "re_test_123"}, nil
}

func (f *fakeStripe) GetPaymentIntent(_ context.Context, id string) (*stripe.PaymentIntent, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	return &stripe.PaymentIntent{ID: id, ClientSecret: id + "_secret_abc"}, nil
}

func (f *fakeStripe) CreateTransfer(_ context.Context, params *stripe.TransferParams) (*stripe.Transfer, error) {
	if f.transferErr != nil {
		return nil, f.transferErr
	}
	f.transfers = append(f.transfers, params)
	return &stripe.Transfer{ID: "tr_test_1"}, nil
}

func (f *fakeStripe) CreateAccount(_ context.Context, params *stripe.AccountParams) (*stripe.Account, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.accounts = append(f.accounts, params)
	return &stripe.Account{ID: "acct_test_1"}, nil
}

func (f *fakeStripe) GetAccount(_ context.Context, id string) (*stripe.Account, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	return &stripe.Account{ID: id, PayoutsEnabled: f.payoutsOn}, nil
}

func (f *fakeStripe) CreateAccountLink(_ context.Context, params *stripe.AccountLinkParams) (*stripe.AccountLink, error) {
	if f.failWith != nil {
		return nil, f.failWith
	}
	f.links = append(f.links, params)
	return &stripe.AccountLink{URL: "https://connect.stripe.com/setup/e/acct_test_1"}, nil
}

// escrowRepoStub is an in-memory EscrowRepository.
type escrowRepoStub struct {
	txs []*domain.EscrowTransaction
}

func (r *escrowRepoStub) CreateTransaction(_ context.Context, tx *domain.EscrowTransaction) error {
	stored := *tx
	r.txs = append(r.txs, &stored)
	return nil
}

func (r *escrowRepoStub) GetTransactionsByTaskID(_ context.Context, taskID uuid.UUID) ([]*domain.EscrowTransaction, error) {
	var out []*domain.EscrowTransaction
	for _, tx := range r.txs {
		if tx.TaskID == taskID {
			out = append(out, tx)
		}
	}
	return out, nil
}

func (r *escrowRepoStub) UpdateTransactionStatus(_ context.Context, id uuid.UUID, status domain.EscrowTransactionStatus) error {
	for _, tx := range r.txs {
		if tx.ID == id {
			tx.Status = status
			return nil
		}
	}
	return sql.ErrNoRows
}

func (r *escrowRepoStub) SetStripePaymentIntent(_ context.Context, id uuid.UUID, paymentIntentID string) error {
	for _, tx := range r.txs {
		if tx.ID == id {
			tx.StripePaymentIntentID = paymentIntentID
			return nil
		}
	}
	return sql.ErrNoRows
}

func (r *escrowRepoStub) GetByStripePaymentIntent(_ context.Context, paymentIntentID string) (*domain.EscrowTransaction, error) {
	for _, tx := range r.txs {
		if tx.StripePaymentIntentID == paymentIntentID {
			return tx, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (r *escrowRepoStub) find(txType domain.EscrowTransactionType) *domain.EscrowTransaction {
	for _, tx := range r.txs {
		if tx.TransactionType == txType {
			return tx
		}
	}
	return nil
}

func stripeFixture(t *testing.T) (*stripeEscrowService, *escrowRepoStub, *fakeStripe, *domain.Task) {
	svc, repo, client, task, _ := stripeFixtureWithUsers(t)
	return svc, repo, client, task
}

// stripeFixtureWithUsers also exposes the user repo, so tests can decide
// whether the claimer has onboarded onto Connect.
func stripeFixtureWithUsers(t *testing.T) (*stripeEscrowService, *escrowRepoStub, *fakeStripe, *domain.Task, *mockUserRepo) {
	t.Helper()

	taskRepo := &mockTaskRepoForClaimSvc{tasks: make(map[uuid.UUID]*domain.Task)}
	task := &domain.Task{ID: uuid.New(), OwnerID: uuid.New(), RewardAmount: 25.50}
	taskRepo.tasks[task.ID] = task

	escrowRepo := &escrowRepoStub{}
	userRepo := &mockUserRepo{stripeAccount: "acct_claimer_1"}
	client := newFakeStripe()
	svc := NewStripeEscrowService(escrowRepo, taskRepo, userRepo, client, "usd").(*stripeEscrowService)
	return svc, escrowRepo, client, task, userRepo
}

func TestLockEscrowAuthorisesWithoutTakingTheMoney(t *testing.T) {
	svc, repo, client, task := stripeFixture(t)

	require.NoError(t, svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))

	require.Len(t, client.created, 1)
	params := client.created[0]
	assert.Equal(t, int64(2550), *params.Amount, "amount must be sent in cents")
	assert.Equal(t, "manual", *params.CaptureMethod, "the money is only held, not captured")
	assert.Equal(t, task.ID.String(), params.Metadata["task_id"])

	lock := repo.find(domain.EscrowTypeLock)
	require.NotNil(t, lock)
	assert.Equal(t, "pi_test_123", lock.StripePaymentIntentID)
	assert.Equal(t, domain.EscrowStatusPending, lock.Status,
		"an authorisation is not a completed payment")
	assert.True(t, task.EscrowLocked)
}

func TestLockEscrowRecordsAFailedPayment(t *testing.T) {
	svc, repo, client, task := stripeFixture(t)
	client.failWith = errors.New("card declined")

	err := svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount)

	require.Error(t, err)
	assert.Equal(t, domain.EscrowStatusFailed, repo.find(domain.EscrowTypeLock).Status)
	assert.False(t, task.EscrowLocked, "a declined card must not lock the task")
}

func TestReleaseEscrowCapturesTheHeldAmount(t *testing.T) {
	svc, repo, client, task := stripeFixture(t)
	require.NoError(t, svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))

	claimerID := uuid.New()
	require.NoError(t, svc.ReleaseEscrow(context.Background(), task.ID, claimerID, task.RewardAmount))

	assert.Equal(t, int64(2550), client.captured["pi_test_123"])
	assert.Equal(t, domain.EscrowStatusCompleted, repo.find(domain.EscrowTypeLock).Status)
	assert.Equal(t, domain.EscrowStatusCompleted, repo.find(domain.EscrowTypeRelease).Status)
}

// Nothing has been taken from an uncaptured hold, so it is cancelled — a
// refund on it would be rejected by Stripe.
func TestRefundCancelsAnUncapturedHold(t *testing.T) {
	svc, _, client, task := stripeFixture(t)
	require.NoError(t, svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))

	require.NoError(t, svc.RefundEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))

	assert.Equal(t, []string{"pi_test_123"}, client.cancelled)
	assert.Empty(t, client.refunded)
	assert.False(t, task.EscrowLocked)
}

func TestRefundIssuesARefundOnceCaptured(t *testing.T) {
	svc, _, client, task := stripeFixture(t)
	require.NoError(t, svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))
	require.NoError(t, svc.ReleaseEscrow(context.Background(), task.ID, uuid.New(), task.RewardAmount))

	require.NoError(t, svc.RefundEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))

	assert.Equal(t, int64(2550), client.refunded["pi_test_123"])
	assert.Empty(t, client.cancelled)
}

func TestCaptureWithoutALockIsRefused(t *testing.T) {
	svc, _, _, task := stripeFixture(t)

	err := svc.ReleaseEscrow(context.Background(), task.ID, uuid.New(), task.RewardAmount)

	assert.ErrorIs(t, err, ErrNoPaymentIntent)
}

func TestLockingTwiceIsRefused(t *testing.T) {
	svc, _, _, task := stripeFixture(t)
	require.NoError(t, svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))

	err := svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount)

	assert.ErrorIs(t, err, ErrEscrowAlreadyLocked)
}

func TestAmountsConvertToCentsWithoutFloatDrift(t *testing.T) {
	// 0.1+0.2 style drift would silently under- or overcharge.
	assert.Equal(t, int64(1010), toMinorUnits(10.10))
	assert.Equal(t, int64(3), toMinorUnits(0.03))
	assert.Equal(t, int64(100000), toMinorUnits(1000))
}

func TestClientSecretGoesOnlyToTheOwner(t *testing.T) {
	svc, _, _, task := stripeFixture(t)
	require.NoError(t, svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))

	secret, err := svc.ClientSecret(context.Background(), task.ID, task.OwnerID)
	require.NoError(t, err)
	assert.Equal(t, "pi_test_123_secret_abc", secret)

	_, err = svc.ClientSecret(context.Background(), task.ID, uuid.New())
	assert.ErrorIs(t, err, ErrUnauthorized, "a stranger must not be able to charge someone else's card")
}

// Approval must move the money on to the claimer's own Stripe account, not
// leave it sitting in the platform balance.
func TestReleasePaysOutToTheClaimersAccount(t *testing.T) {
	svc, repo, client, task, _ := stripeFixtureWithUsers(t)
	require.NoError(t, svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))

	claimerID := uuid.New()
	require.NoError(t, svc.ReleaseEscrow(context.Background(), task.ID, claimerID, task.RewardAmount))

	require.Len(t, client.transfers, 1)
	transfer := client.transfers[0]
	assert.Equal(t, int64(2550), *transfer.Amount)
	assert.Equal(t, "acct_claimer_1", *transfer.Destination)
	assert.Equal(t, "ch_test_1", *transfer.SourceTransaction,
		"the transfer should be tied to the charge it came from")

	payout := repo.find(domain.EscrowTypePayout)
	require.NotNil(t, payout)
	assert.Equal(t, domain.EscrowStatusCompleted, payout.Status)
	assert.Equal(t, claimerID, payout.UserID)
}

// A claimer who has not onboarded yet must still get their task approved; the
// payout waits for them rather than blocking the owner.
func TestReleaseLeavesPayoutPendingWithoutAConnectAccount(t *testing.T) {
	svc, repo, client, task, users := stripeFixtureWithUsers(t)
	users.stripeAccount = ""
	require.NoError(t, svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))

	require.NoError(t, svc.ReleaseEscrow(context.Background(), task.ID, uuid.New(), task.RewardAmount))

	assert.Empty(t, client.transfers)
	assert.Equal(t, int64(2550), client.captured["pi_test_123"], "the money is still captured")

	payout := repo.find(domain.EscrowTypePayout)
	require.NotNil(t, payout)
	assert.Equal(t, domain.EscrowStatusPending, payout.Status)

	pending, err := svc.PendingPayout(context.Background(), task.ID)
	require.NoError(t, err)
	assert.NotNil(t, pending, "the unpaid payout must be findable for a retry")
}

func TestFailedTransferIsRecorded(t *testing.T) {
	svc, repo, client, task, _ := stripeFixtureWithUsers(t)
	require.NoError(t, svc.LockEscrow(context.Background(), task.ID, task.OwnerID, task.RewardAmount))
	client.transferErr = errors.New("destination account restricted")

	err := svc.ReleaseEscrow(context.Background(), task.ID, uuid.New(), task.RewardAmount)

	require.Error(t, err)
	assert.Equal(t, domain.EscrowStatusFailed, repo.find(domain.EscrowTypePayout).Status)
	assert.Equal(t, domain.EscrowStatusCompleted, repo.find(domain.EscrowTypeRelease).Status,
		"the capture succeeded even though the transfer did not")
}

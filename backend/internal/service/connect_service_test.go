package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v82"
)

func TestOnboardingCreatesAnExpressAccountOnce(t *testing.T) {
	users := &mockUserRepo{}
	client := newFakeStripe()
	svc := NewConnectService(users, client)

	url, err := svc.OnboardingLink(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Contains(t, url, "connect.stripe.com")

	require.Len(t, client.accounts, 1)
	assert.Equal(t, string(stripe.AccountTypeExpress), *client.accounts[0].Type)
	assert.True(t, *client.accounts[0].Capabilities.Transfers.Requested,
		"without the transfers capability the account cannot be paid")
	assert.Equal(t, "acct_test_1", users.stripeAccount, "the account must be remembered")

	// Coming back for a second link must reuse the account, not create another.
	_, err = svc.OnboardingLink(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Len(t, client.accounts, 1)
	assert.Len(t, client.links, 2)
}

func TestPayoutsEnabledReflectsStripe(t *testing.T) {
	client := newFakeStripe()

	// Nobody can be paid before they onboard.
	svc := NewConnectService(&mockUserRepo{}, client)
	enabled, err := svc.PayoutsEnabled(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.False(t, enabled)

	// An account exists but Stripe has not finished verifying it.
	svc = NewConnectService(&mockUserRepo{stripeAccount: "acct_test_1"}, client)
	enabled, err = svc.PayoutsEnabled(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.False(t, enabled, "an unverified account must not look payable")

	client.payoutsOn = true
	enabled, err = svc.PayoutsEnabled(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.True(t, enabled)
}

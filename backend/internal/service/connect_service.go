package service

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"github.com/task-underground/backend/internal/repository"
)

// ConnectService onboards claimers onto Stripe Connect so they can be paid
// directly, instead of the money resting in the platform account.
type ConnectService interface {
	// OnboardingLink returns a URL where the user completes Stripe's identity
	// and bank-details flow. Creating the account on first call is idempotent
	// from the caller's point of view.
	OnboardingLink(ctx context.Context, userID uuid.UUID) (string, error)
	// PayoutsEnabled reports whether Stripe will actually send this user money
	// yet — onboarding can be started and abandoned halfway.
	PayoutsEnabled(ctx context.Context, userID uuid.UUID) (bool, error)
}

type connectService struct {
	userRepo   repository.UserRepository
	stripe     StripeAPI
	returnURL  string
	refreshURL string
}

func NewConnectService(userRepo repository.UserRepository, client StripeAPI) ConnectService {
	// Where Stripe sends the user back to. Deep links into the app in a real
	// deployment; a placeholder is fine because Stripe only redirects there.
	base := os.Getenv("STRIPE_CONNECT_RETURN_URL")
	if base == "" {
		base = "https://task-underground.app/connect"
	}
	return &connectService{
		userRepo:   userRepo,
		stripe:     client,
		returnURL:  base + "/done",
		refreshURL: base + "/retry",
	}
}

func (s *connectService) OnboardingLink(ctx context.Context, userID uuid.UUID) (string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	accountID := user.StripeAccountID
	if accountID == "" {
		// Express accounts keep Stripe responsible for the onboarding UI and
		// identity checks, which suits an anonymous marketplace.
		account, err := s.stripe.CreateAccount(ctx, &stripe.AccountParams{
			Type: stripe.String(string(stripe.AccountTypeExpress)),
			Capabilities: &stripe.AccountCapabilitiesParams{
				Transfers: &stripe.AccountCapabilitiesTransfersParams{Requested: stripe.Bool(true)},
			},
			Metadata: map[string]string{"user_id": userID.String()},
		})
		if err != nil {
			return "", fmt.Errorf("create connect account: %w", err)
		}
		accountID = account.ID

		if err := s.userRepo.UpdateStripeAccount(ctx, userID, accountID); err != nil {
			return "", err
		}
	}

	link, err := s.stripe.CreateAccountLink(ctx, &stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		Type:       stripe.String("account_onboarding"),
		ReturnURL:  stripe.String(s.returnURL),
		RefreshURL: stripe.String(s.refreshURL),
	})
	if err != nil {
		return "", fmt.Errorf("create account link: %w", err)
	}
	return link.URL, nil
}

func (s *connectService) PayoutsEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user.StripeAccountID == "" {
		return false, nil
	}

	account, err := s.stripe.GetAccount(ctx, user.StripeAccountID)
	if err != nil {
		return false, fmt.Errorf("retrieve connect account: %w", err)
	}
	return account.PayoutsEnabled, nil
}

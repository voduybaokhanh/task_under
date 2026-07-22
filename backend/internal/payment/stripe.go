// Package payment adapts the Stripe SDK to the narrow interface the escrow
// service needs.
package payment

import (
	"context"
	"log"
	"os"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/accountlink"
	"github.com/stripe/stripe-go/v82/paymentintent"
	"github.com/stripe/stripe-go/v82/refund"
	"github.com/stripe/stripe-go/v82/transfer"
)

// StripeClient wraps the package-level Stripe SDK calls.
type StripeClient struct{}

// NewStripeClient configures the SDK from STRIPE_SECRET_KEY and returns nil
// when the key is missing, so the caller can fall back to simulated escrow.
func NewStripeClient() *StripeClient {
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		log.Println("STRIPE_SECRET_KEY not set; using simulated escrow")
		return nil
	}

	stripe.Key = key

	// STRIPE_API_URL points the SDK at a stub instead of api.stripe.com, for
	// tests and staging.
	if apiURL := os.Getenv("STRIPE_API_URL"); apiURL != "" {
		stripe.SetBackend(stripe.APIBackend, stripe.GetBackendWithConfig(
			stripe.APIBackend, &stripe.BackendConfig{URL: stripe.String(apiURL)}))
		log.Printf("Stripe API pointed at %s", apiURL)
	}

	log.Println("Stripe escrow enabled")
	return &StripeClient{}
}

func (c *StripeClient) CreatePaymentIntent(ctx context.Context, params *stripe.PaymentIntentParams) (*stripe.PaymentIntent, error) {
	params.Context = ctx
	return paymentintent.New(params)
}

func (c *StripeClient) CapturePaymentIntent(ctx context.Context, id string, params *stripe.PaymentIntentCaptureParams) (*stripe.PaymentIntent, error) {
	params.Context = ctx
	return paymentintent.Capture(id, params)
}

func (c *StripeClient) CancelPaymentIntent(ctx context.Context, id string, params *stripe.PaymentIntentCancelParams) (*stripe.PaymentIntent, error) {
	params.Context = ctx
	return paymentintent.Cancel(id, params)
}

func (c *StripeClient) CreateRefund(ctx context.Context, params *stripe.RefundParams) (*stripe.Refund, error) {
	params.Context = ctx
	return refund.New(params)
}

func (c *StripeClient) GetPaymentIntent(ctx context.Context, id string) (*stripe.PaymentIntent, error) {
	return paymentintent.Get(id, &stripe.PaymentIntentParams{Params: stripe.Params{Context: ctx}})
}

func (c *StripeClient) CreateAccount(ctx context.Context, params *stripe.AccountParams) (*stripe.Account, error) {
	params.Context = ctx
	return account.New(params)
}

func (c *StripeClient) GetAccount(ctx context.Context, id string) (*stripe.Account, error) {
	return account.GetByID(id, &stripe.AccountParams{Params: stripe.Params{Context: ctx}})
}

func (c *StripeClient) CreateAccountLink(ctx context.Context, params *stripe.AccountLinkParams) (*stripe.AccountLink, error) {
	params.Context = ctx
	return accountlink.New(params)
}

func (c *StripeClient) CreateTransfer(ctx context.Context, params *stripe.TransferParams) (*stripe.Transfer, error) {
	params.Context = ctx
	return transfer.New(params)
}

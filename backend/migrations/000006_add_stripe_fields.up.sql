-- Stripe references. The PaymentIntent is created with manual capture when a
-- task's escrow is locked, captured on approval and refunded on cancellation.
ALTER TABLE escrow_transactions ADD COLUMN stripe_payment_intent_id TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN stripe_customer_id TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_escrow_stripe_payment_intent
    ON escrow_transactions(stripe_payment_intent_id)
    WHERE stripe_payment_intent_id <> '';

DROP INDEX IF EXISTS idx_escrow_stripe_payment_intent;
ALTER TABLE users DROP COLUMN IF EXISTS stripe_customer_id;
ALTER TABLE escrow_transactions DROP COLUMN IF EXISTS stripe_payment_intent_id;

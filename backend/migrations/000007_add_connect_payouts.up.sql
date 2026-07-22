-- Stripe Connect: claimers are paid out to their own connected account, so the
-- captured money does not sit in the platform balance.
ALTER TABLE users ADD COLUMN stripe_account_id TEXT NOT NULL DEFAULT '';

-- Payouts are tracked like any other escrow movement.
ALTER TABLE escrow_transactions DROP CONSTRAINT escrow_transactions_transaction_type_check;
ALTER TABLE escrow_transactions ADD CONSTRAINT escrow_transactions_transaction_type_check
    CHECK (transaction_type IN ('lock', 'release', 'refund', 'payout'));

DELETE FROM escrow_transactions WHERE transaction_type = 'payout';
ALTER TABLE escrow_transactions DROP CONSTRAINT escrow_transactions_transaction_type_check;
ALTER TABLE escrow_transactions ADD CONSTRAINT escrow_transactions_transaction_type_check
    CHECK (transaction_type IN ('lock', 'release', 'refund'));

ALTER TABLE users DROP COLUMN IF EXISTS stripe_account_id;

DROP TRIGGER IF EXISTS update_chats_updated_at ON chats;
DROP TRIGGER IF EXISTS update_claims_updated_at ON claims;
DROP TRIGGER IF EXISTS update_tasks_updated_at ON tasks;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS arbitrations;
DROP TABLE IF EXISTS escrow_transactions;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS claims;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS users;

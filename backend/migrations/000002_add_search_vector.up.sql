-- Full-text search over tasks (title + description) using PostgreSQL tsvector.
ALTER TABLE tasks ADD COLUMN search_vector tsvector;

CREATE INDEX tasks_search_idx ON tasks USING GIN(search_vector);

-- Backfill existing rows.
UPDATE tasks
SET search_vector = to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''));

-- Keep search_vector in sync on insert/update.
CREATE TRIGGER tasks_search_update
    BEFORE INSERT OR UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION tsvector_update_trigger(search_vector, 'pg_catalog.english', title, description);

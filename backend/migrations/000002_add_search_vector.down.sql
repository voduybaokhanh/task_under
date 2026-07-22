DROP TRIGGER IF EXISTS tasks_search_update ON tasks;
DROP INDEX IF EXISTS tasks_search_idx;
ALTER TABLE tasks DROP COLUMN IF EXISTS search_vector;

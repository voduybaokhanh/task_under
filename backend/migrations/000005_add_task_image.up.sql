-- Public URL of the task's illustration, uploaded straight to S3 by the client.
ALTER TABLE tasks ADD COLUMN image_url TEXT NOT NULL DEFAULT '';

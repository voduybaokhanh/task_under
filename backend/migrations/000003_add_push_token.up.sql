-- Expo push token per user, used to notify a device while the app is closed.
ALTER TABLE users ADD COLUMN push_token TEXT NOT NULL DEFAULT '';

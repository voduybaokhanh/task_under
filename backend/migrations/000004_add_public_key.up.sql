-- X25519 public key (base64) of the user's device, used by other devices to
-- derive a shared secret for end-to-end encrypted chat. The server never sees
-- a private key and cannot read message contents.
ALTER TABLE users ADD COLUMN public_key TEXT NOT NULL DEFAULT '';

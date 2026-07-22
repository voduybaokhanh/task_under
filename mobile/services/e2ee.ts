import nacl from 'tweetnacl';
import { encode as utf8ToBase64, decode as base64ToUtf8, fromUint8Array, toUint8Array } from 'js-base64';

/**
 * Envelope prefix for encrypted messages. Anything without it is legacy
 * plaintext written before E2EE existed, and is shown as-is.
 */
export const ENVELOPE_PREFIX = 'E2E1.';

// js-base64 handles UTF-8 for us, which avoids depending on TextEncoder —
// Hermes does not ship TextDecoder.
const toBytes = (text: string) => toUint8Array(utf8ToBase64(text));
const toText = (bytes: Uint8Array) => base64ToUtf8(fromUint8Array(bytes));

export function isEncrypted(payload: string): boolean {
  return payload.startsWith(ENVELOPE_PREFIX);
}

/**
 * Derives the secret shared by two devices: nacl.box.before is symmetric, so
 * both sides compute the same key and either can decrypt the conversation.
 */
export function deriveSharedKey(theirPublicKeyBase64: string, mySecretKey: Uint8Array): Uint8Array {
  return nacl.box.before(toUint8Array(theirPublicKeyBase64), mySecretKey);
}

/** Encrypts to `E2E1.<nonce>.<ciphertext>`, both base64. */
export function encrypt(plaintext: string, sharedKey: Uint8Array): string {
  const nonce = nacl.randomBytes(nacl.box.nonceLength);
  const box = nacl.box.after(toBytes(plaintext), nonce, sharedKey);
  return `${ENVELOPE_PREFIX}${fromUint8Array(nonce)}.${fromUint8Array(box)}`;
}

/**
 * Reverses encrypt(). Returns null when the payload is malformed or the key is
 * wrong, so callers can fall back rather than crash a whole conversation.
 */
export function decrypt(payload: string, sharedKey: Uint8Array): string | null {
  if (!isEncrypted(payload)) {
    return null;
  }

  const [nonceBase64, boxBase64] = payload.slice(ENVELOPE_PREFIX.length).split('.');
  if (!nonceBase64 || !boxBase64) {
    return null;
  }

  try {
    const opened = nacl.box.open.after(toUint8Array(boxBase64), toUint8Array(nonceBase64), sharedKey);
    return opened ? toText(opened) : null;
  } catch {
    return null;
  }
}

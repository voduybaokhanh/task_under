import * as Crypto from 'expo-crypto';
import * as SecureStore from 'expo-secure-store';
import nacl from 'tweetnacl';
import { fromUint8Array, toUint8Array } from 'js-base64';

import { apiService } from './api';

const SECRET_KEY_STORE = 'e2ee_secret_key';

// tweetnacl looks for a browser/Node CSPRNG that React Native does not have,
// so point it at the platform's own secure random source.
nacl.setPRNG((bytes, length) => {
  bytes.set(Crypto.getRandomBytes(length));
});

let cached: nacl.BoxKeyPair | null = null;

/**
 * Returns this device's X25519 key pair, generating it on first run. The
 * secret key lives in the OS keystore (Keychain / Android Keystore) and never
 * leaves the device — that is what makes the chat end-to-end encrypted.
 */
export async function getKeyPair(): Promise<nacl.BoxKeyPair> {
  if (cached) {
    return cached;
  }

  const stored = await SecureStore.getItemAsync(SECRET_KEY_STORE);
  if (stored) {
    cached = nacl.box.keyPair.fromSecretKey(toUint8Array(stored));
    return cached;
  }

  const keyPair = nacl.box.keyPair();
  await SecureStore.setItemAsync(SECRET_KEY_STORE, fromUint8Array(keyPair.secretKey));
  cached = keyPair;
  return keyPair;
}

/**
 * Publishes the public half so other users can encrypt to this device. Safe to
 * call on every launch; the backend just overwrites the same value.
 */
export async function publishPublicKey(): Promise<void> {
  const { publicKey } = await getKeyPair();
  await apiService.updatePublicKey(fromUint8Array(publicKey));
}

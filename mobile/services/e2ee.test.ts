// Run with: npm test  (node --test, no test framework needed)
import assert from 'node:assert/strict';
import { test } from 'node:test';

import nacl from 'tweetnacl';
import { fromUint8Array } from 'js-base64';

import { decrypt, deriveSharedKey, encrypt, isEncrypted } from './e2ee.ts';

const alice = nacl.box.keyPair();
const bob = nacl.box.keyPair();

const aliceToBob = deriveSharedKey(fromUint8Array(bob.publicKey), alice.secretKey);
const bobToAlice = deriveSharedKey(fromUint8Array(alice.publicKey), bob.secretKey);

test('both sides derive the same shared key', () => {
  assert.deepEqual(aliceToBob, bobToAlice);
});

test('the recipient reads what the sender wrote', () => {
  const envelope = encrypt('gặp ở quán cũ lúc 8h', aliceToBob);

  assert.ok(isEncrypted(envelope));
  assert.equal(decrypt(envelope, bobToAlice), 'gặp ở quán cũ lúc 8h');
});

test('the sender can still read their own message', () => {
  const envelope = encrypt('mine to keep', aliceToBob);

  assert.equal(decrypt(envelope, aliceToBob), 'mine to keep');
});

test('the ciphertext leaks neither the plaintext nor a repeated nonce', () => {
  const first = encrypt('secret', aliceToBob);
  const second = encrypt('secret', aliceToBob);

  assert.ok(!first.includes('secret'));
  assert.notEqual(first, second, 'same plaintext twice must not produce the same envelope');
});

test('a third party holding neither key gets nothing', () => {
  const mallory = nacl.box.keyPair();
  const wrongKey = deriveSharedKey(fromUint8Array(alice.publicKey), mallory.secretKey);

  assert.equal(decrypt(encrypt('private', aliceToBob), wrongKey), null);
});

test('tampering with the ciphertext is detected', () => {
  const envelope = encrypt('transfer 10', aliceToBob);
  const tampered = envelope.slice(0, -2) + (envelope.endsWith('AA') ? 'BB' : 'AA');

  assert.equal(decrypt(tampered, bobToAlice), null);
});

test('legacy plaintext is recognised as unencrypted', () => {
  assert.equal(isEncrypted('hello from before E2EE'), false);
  assert.equal(decrypt('hello from before E2EE', aliceToBob), null);
});

test('a malformed envelope returns null instead of throwing', () => {
  assert.equal(decrypt('E2E1.only-one-part', aliceToBob), null);
  assert.equal(decrypt('E2E1..', aliceToBob), null);
});

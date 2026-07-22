// Manual end-to-end check against a running backend:
//   DATABASE_URL-backed server on :8080, then
//   node --test services/e2ee.e2e.ts   (or: npm run test:e2e)
//
// Two simulated devices exchange an encrypted message and assert that what the
// server hands back is ciphertext, while the recipient still reads the words.
import assert from 'node:assert/strict';
import { test } from 'node:test';

import nacl from 'tweetnacl';
import { fromUint8Array } from 'js-base64';

import { decrypt, deriveSharedKey, encrypt, isEncrypted } from './e2ee.ts';

const BACKEND = process.env.E2E_BACKEND ?? 'http://localhost:8080';

async function call(method: string, path: string, device: string, body?: unknown) {
  const response = await fetch(BACKEND + path, {
    method,
    headers: { 'X-Device-ID': device, 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  assert.ok(response.ok, `${method} ${path} -> ${response.status}: ${text}`);
  return text ? JSON.parse(text) : null;
}

test('the server stores ciphertext the recipient alone can read', async () => {
  const stamp = Date.now();
  const aliceDevice = `e2ee-alice-${stamp}`;
  const bobDevice = `e2ee-bob-${stamp}`;

  const alice = nacl.box.keyPair();
  const bob = nacl.box.keyPair();

  await call('PUT', '/api/v1/users/me/pubkey', aliceDevice, {
    public_key: fromUint8Array(alice.publicKey),
  });
  await call('PUT', '/api/v1/users/me/pubkey', bobDevice, {
    public_key: fromUint8Array(bob.publicKey),
  });

  const aliceUser = await call('GET', '/api/v1/users/me', aliceDevice);

  const task = await call('POST', '/api/v1/tasks', aliceDevice, {
    title: 'e2ee task',
    description: 'encrypted chat check',
    reward_amount: 10,
    max_claimants: 1,
    claim_deadline: new Date(Date.now() + 3600_000).toISOString(),
    owner_deadline: new Date(Date.now() + 7200_000).toISOString(),
  });

  // Bob opens the chat and looks Alice's key up the way the app does.
  const chat = await call('POST', `/api/v1/tasks/${task.id}/chats`, bobDevice);
  const { public_key: alicePublished } = await call(
    'GET',
    `/api/v1/users/${aliceUser.id}/pubkey`,
    bobDevice,
  );
  assert.equal(alicePublished, fromUint8Array(alice.publicKey));

  const plaintext = 'toạ độ: gốc cây thứ ba';
  const bobToAlice = deriveSharedKey(alicePublished, bob.secretKey);
  await call('POST', `/api/v1/chats/${chat.id}/messages`, bobDevice, {
    content: encrypt(plaintext, bobToAlice),
  });

  const { messages } = await call('GET', `/api/v1/chats/${chat.id}/messages`, aliceDevice);
  const stored = messages[0].content;

  assert.ok(isEncrypted(stored), 'the server returned something that is not an envelope');
  assert.ok(!stored.includes('toạ độ'), 'plaintext leaked into storage');

  const aliceToBob = deriveSharedKey(fromUint8Array(bob.publicKey), alice.secretKey);
  assert.equal(decrypt(stored, aliceToBob), plaintext);

  console.log('stored on the server:', stored);
});

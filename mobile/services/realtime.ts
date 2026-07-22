import { Message } from '../types';
import { apiService } from './api';
import { WebSocketService } from './websocket';
import { useChatStore } from '../store/useChatStore';

let socket: WebSocketService | null = null;

/**
 * Opens the app's single WebSocket connection and routes incoming events into
 * the stores. Chat messages arrive as ciphertext and are decrypted by the chat
 * store, same as messages fetched over HTTP.
 */
export async function connectRealtime(): Promise<void> {
  if (socket) {
    return;
  }

  const deviceId = await apiService.getDeviceId();
  if (!deviceId) {
    return;
  }

  const baseURL = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080';
  socket = new WebSocketService(baseURL, deviceId);

  socket.on('chat_message', (message: Message) => {
    const { selectedChat, addMessage } = useChatStore.getState();
    // Only the open conversation has a shared key, so anything else would be
    // stored as undecryptable ciphertext.
    if (selectedChat?.id === message.chat_id) {
      addMessage(message.chat_id, message);
    }
  });

  await socket.connect();
}

export function disconnectRealtime(): void {
  socket?.disconnect();
  socket = null;
}

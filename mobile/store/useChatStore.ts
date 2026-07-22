import { create } from 'zustand';
import { Chat, Message } from '../types';
import { apiService } from '../services/api';
import { decrypt, encrypt, isEncrypted, deriveSharedKey } from '../services/e2ee';
import { getKeyPair } from '../services/keys';

interface ChatState {
  chats: Chat[];
  messages: Map<string, Message[]>;
  selectedChat: Chat | null;
  /** Our own user ID, needed to tell our messages from theirs. */
  myUserId: string | null;
  /** Set once a shared secret with the other participant exists. */
  encrypted: boolean;
  loading: boolean;
  error: string | null;

  fetchChats: (taskId: string) => Promise<void>;
  getOrCreateChat: (taskId: string, claimerId?: string) => Promise<void>;
  deleteChat: (chatId: string) => Promise<void>;
  sendMessage: (chatId: string, content: string) => Promise<void>;
  fetchMessages: (chatId: string) => Promise<void>;
  setSelectedChat: (chat: Chat | null) => void;
  addMessage: (chatId: string, message: Message) => void;
  clearError: () => void;
}

// The key shared with the current conversation partner. Kept outside the
// store so key material never lands in React state or a devtools snapshot.
let sharedKey: Uint8Array | null = null;

/**
 * Establishes the shared secret for a chat. Falls back to plaintext when the
 * other side has not published a public key yet (an older app version).
 */
async function openSecureChannel(chat: Chat, myUserId: string): Promise<boolean> {
  sharedKey = null;

  const otherUserId =
    chat.participant_id === myUserId ? chat.other_participant_id : chat.participant_id;

  const theirPublicKey = await apiService.getPublicKey(otherUserId);
  if (!theirPublicKey) {
    return false;
  }

  const { secretKey } = await getKeyPair();
  sharedKey = deriveSharedKey(theirPublicKey, secretKey);
  return true;
}

/** Replaces each ciphertext with its plaintext, in place of the stored value. */
function decryptMessages(messages: Message[]): Message[] {
  return messages.map((message) => {
    if (!sharedKey || !isEncrypted(message.content)) {
      return message;
    }
    const plaintext = decrypt(message.content, sharedKey);
    return { ...message, content: plaintext ?? '🔒 Không giải mã được tin nhắn này' };
  });
}

export const useChatStore = create<ChatState>((set, get) => ({
  chats: [],
  messages: new Map(),
  selectedChat: null,
  myUserId: null,
  encrypted: false,
  loading: false,
  error: null,

  fetchChats: async (taskId: string) => {
    set({ loading: true, error: null });
    try {
      const chats = await apiService.getChats(taskId);
      set({ chats, loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  getOrCreateChat: async (taskId: string, claimerId?: string) => {
    set({ loading: true, error: null });
    try {
      const chat = await apiService.getOrCreateChat(taskId, claimerId);
      const me = await apiService.getMe();
      const encrypted = await openSecureChannel(chat, me.id);

      set({ selectedChat: chat, myUserId: me.id, encrypted, loading: false });
      await get().fetchMessages(chat.id);
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  deleteChat: async (chatId: string) => {
    set({ loading: true, error: null });
    try {
      await apiService.deleteChat(chatId);
      sharedKey = null;
      set({ selectedChat: null, encrypted: false, loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  sendMessage: async (chatId: string, content: string) => {
    set({ loading: true, error: null });
    try {
      const wire = sharedKey ? encrypt(content, sharedKey) : content;
      const message = await apiService.sendMessage(chatId, wire);

      // Show what the user typed, not what went over the wire.
      get().addMessage(chatId, { ...message, content });
      set({ loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  fetchMessages: async (chatId: string) => {
    set({ loading: true, error: null });
    try {
      const msgs = await apiService.getMessages(chatId);
      const messages = new Map(get().messages);
      messages.set(chatId, decryptMessages(msgs));
      set({ messages, loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  setSelectedChat: (chat: Chat | null) => {
    set({ selectedChat: chat });
    if (chat) {
      get().fetchMessages(chat.id);
    }
  },

  addMessage: (chatId: string, message: Message) => {
    const messages = new Map(get().messages);
    const existing = messages.get(chatId) || [];
    messages.set(chatId, [...existing, ...decryptMessages([message])]);
    set({ messages });
  },

  clearError: () => {
    set({ error: null });
  },
}));

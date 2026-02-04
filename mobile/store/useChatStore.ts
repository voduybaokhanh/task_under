import { create } from 'zustand';
import { Chat, Message } from '../types';
import { apiService } from '../services/api';

interface ChatState {
  chats: Chat[];
  messages: Map<string, Message[]>;
  selectedChat: Chat | null;
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

export const useChatStore = create<ChatState>((set, get) => ({
  chats: [],
  messages: new Map(),
  selectedChat: null,
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
      set({ selectedChat: chat, loading: false });
      await get().fetchMessages(chat.id);
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  deleteChat: async (chatId: string) => {
    set({ loading: true, error: null });
    try {
      await apiService.deleteChat(chatId);
      set({ selectedChat: null, loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  sendMessage: async (chatId: string, content: string) => {
    set({ loading: true, error: null });
    try {
      const message = await apiService.sendMessage(chatId, content);
      get().addMessage(chatId, message);
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
      messages.set(chatId, msgs);
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
    messages.set(chatId, [...existing, message]);
    set({ messages });
  },

  clearError: () => {
    set({ error: null });
  },
}));

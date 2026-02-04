import axios, { AxiosInstance } from 'axios';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { Task, Claim, Chat, Message } from '../types';

const DEVICE_ID_KEY = 'device_id';

class ApiService {
  private client: AxiosInstance;
  private baseURL: string;

  constructor() {
    this.baseURL = process.env.EXPO_PUBLIC_API_URL || 'http://localhost:8080';
    this.client = axios.create({
      baseURL: this.baseURL,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Request interceptor to add device ID
    this.client.interceptors.request.use(async (config) => {
      const deviceId = await this.getDeviceId();
      if (deviceId) {
        config.headers['X-Device-ID'] = deviceId;
      }
      return config;
    });
  }

  private async getDeviceId(): Promise<string | null> {
    let deviceId = await AsyncStorage.getItem(DEVICE_ID_KEY);
    if (!deviceId) {
      deviceId = this.generateDeviceId();
      await AsyncStorage.setItem(DEVICE_ID_KEY, deviceId);
    }
    return deviceId;
  }

  private generateDeviceId(): string {
    // Generate a simple device ID (in production, use a proper UUID library)
    return `device_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  // Task endpoints
  async createTask(data: {
    title: string;
    description: string;
    reward_amount: number;
    max_claimants: number;
    claim_deadline: string;
    owner_deadline: string;
  }): Promise<Task> {
    const response = await this.client.post<Task>('/api/v1/tasks', data);
    return response.data;
  }

  async getOpenTasks(limit = 20, offset = 0): Promise<Task[]> {
    const response = await this.client.get<{ tasks: Task[] }>('/api/v1/tasks', {
      params: { limit, offset },
    });
    return response.data.tasks;
  }

  async getTask(id: string): Promise<Task> {
    const response = await this.client.get<Task>(`/api/v1/tasks/${id}`);
    return response.data;
  }

  async getUserTasks(limit = 20, offset = 0): Promise<Task[]> {
    const response = await this.client.get<{ tasks: Task[] }>('/api/v1/tasks/my', {
      params: { limit, offset },
    });
    return response.data.tasks;
  }

  // Claim endpoints
  async claimTask(taskId: string): Promise<Claim> {
    const response = await this.client.post<Claim>(`/api/v1/tasks/${taskId}/claims`);
    return response.data;
  }

  async getClaimsByTask(taskId: string): Promise<Claim[]> {
    const response = await this.client.get<{ claims: Claim[] }>(`/api/v1/tasks/${taskId}/claims`);
    return response.data.claims;
  }

  async getClaim(id: string): Promise<Claim> {
    const response = await this.client.get<Claim>(`/api/v1/claims/${id}`);
    return response.data;
  }

  async submitCompletion(claimId: string, text: string, imageUrl?: string): Promise<Claim> {
    const response = await this.client.post<Claim>(`/api/v1/claims/${claimId}/submit`, {
      text,
      image_url: imageUrl,
    });
    return response.data;
  }

  async approveClaim(claimId: string): Promise<void> {
    await this.client.post(`/api/v1/claims/${claimId}/approve`);
  }

  async rejectClaim(claimId: string): Promise<void> {
    await this.client.post(`/api/v1/claims/${claimId}/reject`);
  }

  // Chat endpoints
  async getChats(taskId: string): Promise<Chat[]> {
    const response = await this.client.get<{ chats: Chat[] }>(`/api/v1/tasks/${taskId}/chats`);
    return response.data.chats;
  }

  async getOrCreateChat(taskId: string, claimerId?: string): Promise<Chat> {
    const params = claimerId ? { claimer_id: claimerId } : {};
    const response = await this.client.post<Chat>(`/api/v1/tasks/${taskId}/chats`, {}, { params });
    return response.data;
  }

  async deleteChat(chatId: string): Promise<void> {
    await this.client.delete(`/api/v1/chats/${chatId}`);
  }

  async sendMessage(chatId: string, content: string): Promise<Message> {
    const response = await this.client.post<Message>(`/api/v1/chats/${chatId}/messages`, {
      content,
    });
    return response.data;
  }

  async getMessages(chatId: string, limit = 50, offset = 0): Promise<Message[]> {
    const response = await this.client.get<{ messages: Message[] }>(
      `/api/v1/chats/${chatId}/messages`,
      {
        params: { limit, offset },
      }
    );
    return response.data.messages;
  }
}

export const apiService = new ApiService();

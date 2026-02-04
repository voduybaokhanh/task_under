import { create } from 'zustand';
import { Task, Claim } from '../types';
import { apiService } from '../services/api';

interface TaskState {
  tasks: Task[];
  myTasks: Task[];
  selectedTask: Task | null;
  claims: Claim[];
  loading: boolean;
  error: string | null;
  
  fetchOpenTasks: () => Promise<void>;
  fetchMyTasks: () => Promise<void>;
  fetchTask: (id: string) => Promise<void>;
  createTask: (data: {
    title: string;
    description: string;
    reward_amount: number;
    max_claimants: number;
    claim_deadline: string;
    owner_deadline: string;
  }) => Promise<void>;
  claimTask: (taskId: string) => Promise<void>;
  fetchClaims: (taskId: string) => Promise<void>;
  submitCompletion: (claimId: string, text: string, imageUrl?: string) => Promise<void>;
  approveClaim: (claimId: string) => Promise<void>;
  rejectClaim: (claimId: string) => Promise<void>;
  setSelectedTask: (task: Task | null) => void;
  clearError: () => void;
}

export const useTaskStore = create<TaskState>((set, get) => ({
  tasks: [],
  myTasks: [],
  selectedTask: null,
  claims: [],
  loading: false,
  error: null,

  fetchOpenTasks: async () => {
    set({ loading: true, error: null });
    try {
      const tasks = await apiService.getOpenTasks();
      set({ tasks, loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  fetchMyTasks: async () => {
    set({ loading: true, error: null });
    try {
      const tasks = await apiService.getUserTasks();
      set({ myTasks: tasks, loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  fetchTask: async (id: string) => {
    set({ loading: true, error: null });
    try {
      const task = await apiService.getTask(id);
      set({ selectedTask: task, loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  createTask: async (data) => {
    set({ loading: true, error: null });
    try {
      await apiService.createTask(data);
      await get().fetchOpenTasks();
      await get().fetchMyTasks();
      set({ loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  claimTask: async (taskId: string) => {
    set({ loading: true, error: null });
    try {
      await apiService.claimTask(taskId);
      await get().fetchTask(taskId);
      await get().fetchClaims(taskId);
      set({ loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  fetchClaims: async (taskId: string) => {
    set({ loading: true, error: null });
    try {
      const claims = await apiService.getClaimsByTask(taskId);
      set({ claims, loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  submitCompletion: async (claimId: string, text: string, imageUrl?: string) => {
    set({ loading: true, error: null });
    try {
      await apiService.submitCompletion(claimId, text, imageUrl);
      const claim = await apiService.getClaim(claimId);
      const taskId = claim.task_id;
      await get().fetchTask(taskId);
      await get().fetchClaims(taskId);
      set({ loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  approveClaim: async (claimId: string) => {
    set({ loading: true, error: null });
    try {
      await apiService.approveClaim(claimId);
      const claim = await apiService.getClaim(claimId);
      const taskId = claim.task_id;
      await get().fetchTask(taskId);
      await get().fetchClaims(taskId);
      set({ loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  rejectClaim: async (claimId: string) => {
    set({ loading: true, error: null });
    try {
      await apiService.rejectClaim(claimId);
      const claim = await apiService.getClaim(claimId);
      const taskId = claim.task_id;
      await get().fetchTask(taskId);
      await get().fetchClaims(taskId);
      set({ loading: false });
    } catch (error: any) {
      set({ error: error.message, loading: false });
    }
  },

  setSelectedTask: (task: Task | null) => {
    set({ selectedTask: task });
  },

  clearError: () => {
    set({ error: null });
  },
}));

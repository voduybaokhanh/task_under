import { create } from 'zustand';
import { User } from '../types';
import { apiService } from '../services/api';

interface UserState {
  me: User | null;
  loading: boolean;
  fetchMe: () => Promise<void>;
}

export const useUserStore = create<UserState>((set) => ({
  me: null,
  loading: false,

  fetchMe: async () => {
    set({ loading: true });
    try {
      const me = await apiService.getMe();
      set({ me, loading: false });
    } catch {
      set({ loading: false });
    }
  },
}));

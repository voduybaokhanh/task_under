export interface User {
  id: string;
  device_id: string;
  created_at: string;
  reputation: number;
  total_earned: number;
  total_spent: number;
}

export interface Task {
  id: string;
  owner_id: string;
  title: string;
  description: string;
  reward_amount: number;
  max_claimants: number;
  claim_deadline: string;
  owner_deadline: string;
  status: 'open' | 'claimed' | 'completed' | 'cancelled' | 'disputed';
  escrow_locked: boolean;
  created_at: string;
  updated_at: string;
}

export interface Claim {
  id: string;
  task_id: string;
  claimer_id: string;
  status: 'pending' | 'approved' | 'rejected' | 'cancelled';
  submitted_at?: string;
  completion_text?: string;
  completion_image_url?: string;
  created_at: string;
  updated_at: string;
}

export interface Chat {
  id: string;
  task_id: string;
  participant_id: string;
  other_participant_id: string;
  deleted_by_participant: boolean;
  deleted_by_other: boolean;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: string;
  chat_id: string;
  sender_id: string;
  content: string;
  created_at: string;
}

export interface ApiResponse<T> {
  data?: T;
  error?: string;
}

export type UserRole = 'superadmin' | 'admin' | 'moderator';

export interface AdminUser {
  id: number;
  username: string;
  role: UserRole;
  display_name: string;
  active: boolean;
  created_at: string;
  updated_at: string;
  last_login: string | null;
}

export interface Channel {
  id: number;
  gid: string;
  telegram_id: number;
  name: string;
  username: string;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ChannelSettings {
  id: number;
  gid: string;
  similarity_threshold: number;
  min_msg_len: number;
  max_emoji: number;
  min_spam_probability: number;
  first_messages_count: number;
  paranoid_mode: boolean;
  cas_enabled: boolean;
  openai_enabled: boolean;
  openai_model: string;
  openai_veto: boolean;
  meta_links_limit: number;
  meta_links_only: boolean;
  meta_image_only: boolean;
  meta_video_only: boolean;
  meta_audio_only: boolean;
  meta_contact_only: boolean;
  meta_forward: boolean;
  meta_keyboard: boolean;
  meta_username_symbols: boolean;
  meta_giveaway: boolean;
  duplicates_threshold: number;
  duplicates_window: string;
  training_mode: boolean;
  dry_mode: boolean;
  soft_ban: boolean;
  no_spam_reply: boolean;
  aggressive_cleanup: boolean;
  aggressive_cleanup_limit: number;
  suppress_join_message: boolean;
  delete_join_messages: boolean;
  delete_leave_messages: boolean;
}

export interface SpamCheck {
  name: string;
  spam: boolean;
  details: string;
}

export interface DetectedSpamEntry {
  id: number;
  gid: string;
  text: string;
  user_id: number;
  user_name: string;
  timestamp: string;
  added: boolean;
  checks: SpamCheck[];
}

export interface ApprovedUser {
  user_id: string;
  user_name: string;
  timestamp: string;
}

export interface DashboardStats {
  total_spam: number;
  total_approved: number;
  active_channels: number;
  spam_by_type: Record<string, number>;
  spam_timeline: { date: string; count: number }[];
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface DictionaryEntry {
  type: 'stop_phrase' | 'ignored_word';
  data: string;
}

export interface SampleEntry {
  type: 'spam' | 'ham';
  origin: 'preset' | 'user';
  message: string;
}

import apiClient from './client';
import type { DetectedSpamEntry, SpamCheck } from '@/types';

interface DetectedSpamResponse {
  entries: DetectedSpamEntry[];
  total: number;
  page: number;
  per_page: number;
}

interface SpamCheckResponse {
  spam: boolean;
  checks: SpamCheck[];
}

export const spamApi = {
  getDetected: (gid: string, page = 1, perPage = 50, from?: string, to?: string) =>
    apiClient.get<DetectedSpamResponse>('/spam/detected', {
      params: { gid, page, per_page: perPage, from, to },
    }),

  addToSamples: (id: number, msg: string) =>
    apiClient.post(`/spam/detected/${id}/add`, { msg }),

  unban: (gid: string, userId: number) =>
    apiClient.post('/spam/unban', { gid, user_id: userId }),

  check: (text: string, gid?: string) =>
    apiClient.post<SpamCheckResponse>('/spam/check', { msg: text, check_only: true, gid }),
};

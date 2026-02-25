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
  getDetected: (gid: string, page = 1, perPage = 50) =>
    apiClient.get<DetectedSpamResponse>('/spam/detected', {
      params: { gid, page, per_page: perPage },
    }),

  addToSamples: (id: number, msg: string) =>
    apiClient.post(`/spam/detected/${id}/add`, { msg }),

  check: (text: string) =>
    apiClient.post<SpamCheckResponse>('/spam/check', { msg: text, check_only: true }),
};

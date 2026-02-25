import apiClient from './client';
import type { DetectedSpamEntry, SpamCheck } from '@/types';

export const spamApi = {
  getDetected: (gid: string, page = 1, limit = 50) =>
    apiClient.get<DetectedSpamEntry[]>('/spam/detected', {
      params: { gid, page, limit },
    }),

  addToSamples: (id: number) =>
    apiClient.post(`/spam/detected/${id}/add`),

  check: (text: string, gid: string) =>
    apiClient.post<SpamCheck[]>('/spam/check', { text, gid }),
};

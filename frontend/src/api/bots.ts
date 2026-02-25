import apiClient from './client';
import type { Bot } from '@/types';

export const botsApi = {
  list: () =>
    apiClient.get<Bot[]>('/bots'),

  create: (data: { name: string; token: string }) =>
    apiClient.post<{ id: number }>('/bots', data),

  update: (id: number, data: Partial<Bot>) =>
    apiClient.put(`/bots/${id}`, data),

  delete: (id: number) =>
    apiClient.delete(`/bots/${id}`),

  validate: (id: number) =>
    apiClient.post<{ valid: boolean; username?: string; error?: string }>(`/bots/${id}/validate`),
};

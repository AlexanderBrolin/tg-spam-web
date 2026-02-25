import apiClient from './client';
import type { DashboardStats } from '@/types';

export const statsApi = {
  get: (gid?: string) =>
    apiClient.get<DashboardStats>('/stats', { params: gid ? { gid } : {} }),
};

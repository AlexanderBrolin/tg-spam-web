import apiClient from './client';
import type { AdminUser } from '@/types';

export const adminUsersApi = {
  list: () =>
    apiClient.get<AdminUser[]>('/admin/users/'),

  create: (data: { username: string; password: string; role: string; display_name: string }) =>
    apiClient.post<AdminUser>('/admin/users/', data),

  update: (id: number, data: Partial<AdminUser>) =>
    apiClient.put<AdminUser>(`/admin/users/${id}`, data),

  delete: (id: number) =>
    apiClient.delete(`/admin/users/${id}`),
};

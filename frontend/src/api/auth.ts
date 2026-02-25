import apiClient from './client';
import type { LoginRequest, TokenPair, AdminUser } from '@/types';

export const authApi = {
  login: (data: LoginRequest) =>
    apiClient.post<TokenPair>('/auth/login', data),

  refresh: (refreshToken: string) =>
    apiClient.post<TokenPair>('/auth/refresh', { refresh_token: refreshToken }),

  me: () =>
    apiClient.get<AdminUser>('/auth/me'),

  changePassword: (oldPassword: string, newPassword: string) =>
    apiClient.put('/auth/password', { old_password: oldPassword, new_password: newPassword }),

  logout: (refreshToken: string) =>
    apiClient.post('/auth/logout', { refresh_token: refreshToken }),
};

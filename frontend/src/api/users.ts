import apiClient from './client';
import type { ApprovedUser } from '@/types';

interface ApprovedUsersResponse {
  users: ApprovedUser[];
  total: number;
}

export const usersApi = {
  getApproved: (gid: string) =>
    apiClient.get<ApprovedUsersResponse>('/users/approved', { params: { gid } }),

  addApproved: (gid: string, userId: string, userName: string) =>
    apiClient.post('/users/approved', { gid, user_id: userId, user_name: userName }),

  removeApproved: (gid: string, userId: string) =>
    apiClient.delete(`/users/approved/${userId}`, { params: { gid } }),
};

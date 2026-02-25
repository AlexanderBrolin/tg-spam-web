import apiClient from './client';
import type { Channel, ChannelSettings } from '@/types';

export const channelsApi = {
  list: () =>
    apiClient.get<Channel[]>('/channels/'),

  get: (gid: string) =>
    apiClient.get<Channel>(`/channels/${gid}`),

  create: (data: Partial<Channel>) =>
    apiClient.post<Channel>('/channels/', data),

  update: (id: number, data: Partial<Channel>) =>
    apiClient.put<Channel>(`/channels/${id}`, data),

  delete: (id: number) =>
    apiClient.delete(`/channels/${id}`),

  getSettings: (gid: string) =>
    apiClient.get<ChannelSettings>(`/channels/${gid}/settings`),

  updateSettings: (gid: string, data: Partial<ChannelSettings>) =>
    apiClient.put(`/channels/${gid}/settings`, data),
};

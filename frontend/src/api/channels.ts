import apiClient from './client';
import type { Channel, ChannelSettings } from '@/types';

export const channelsApi = {
  list: () =>
    apiClient.get<Channel[]>('/channels'),

  get: (gid: string) =>
    apiClient.get<Channel>(`/channels/${gid}`),

  create: (data: Partial<Channel>) =>
    apiClient.post<Channel>('/channels', data),

  update: (gid: string, data: Partial<Channel>) =>
    apiClient.put<Channel>(`/channels/${gid}`, data),

  delete: (gid: string) =>
    apiClient.delete(`/channels/${gid}`),

  getSettings: (gid: string) =>
    apiClient.get<ChannelSettings>(`/channels/${gid}/settings`),

  updateSettings: (gid: string, data: Partial<ChannelSettings>) =>
    apiClient.put(`/channels/${gid}/settings`, data),
};

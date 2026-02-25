import apiClient from './client';
import type { DictionaryEntry } from '@/types';

export const dictionaryApi = {
  get: (gid: string) =>
    apiClient.get<DictionaryEntry[]>('/dictionary', { params: { gid } }),

  add: (gid: string, type: string, data: string) =>
    apiClient.post('/dictionary', { gid, type, data }),

  remove: (gid: string, type: string, data: string) =>
    apiClient.delete('/dictionary', { data: { gid, type, data } }),
};

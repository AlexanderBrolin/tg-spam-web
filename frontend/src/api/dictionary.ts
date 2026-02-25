import apiClient from './client';
import type { DictionaryEntry } from '@/types';

interface DictionaryResponse {
  stop_phrases: DictionaryEntry[];
  ignored_words: DictionaryEntry[];
  stats: { stop_phrases: number; ignored_words: number } | null;
}

export const dictionaryApi = {
  get: () =>
    apiClient.get<DictionaryResponse>('/dictionary/'),

  add: (type: string, data: string) =>
    apiClient.post('/dictionary/', { type, data }),

  remove: (id: number) =>
    apiClient.delete('/dictionary/', { params: { id } }),
};

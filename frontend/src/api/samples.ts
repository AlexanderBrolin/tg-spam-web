import apiClient from './client';

interface SamplesResponse {
  spam: string[];
  ham: string[];
  spam_count: number;
  ham_count: number;
}

export const samplesApi = {
  getAll: () =>
    apiClient.get<SamplesResponse>('/samples/'),

  addSpam: (message: string) =>
    apiClient.post('/samples/spam', { msg: message }),

  addHam: (message: string) =>
    apiClient.post('/samples/ham', { msg: message }),

  deleteSpam: (message: string) =>
    apiClient.delete('/samples/spam', { data: { msg: message } }),

  deleteHam: (message: string) =>
    apiClient.delete('/samples/ham', { data: { msg: message } }),

  reload: () =>
    apiClient.put('/samples/reload'),
};

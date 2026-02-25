import apiClient from './client';

export const samplesApi = {
  getSpam: (gid: string) =>
    apiClient.get<string[]>('/samples/spam', { params: { gid } }),

  getHam: (gid: string) =>
    apiClient.get<string[]>('/samples/ham', { params: { gid } }),

  addSpam: (gid: string, message: string) =>
    apiClient.post('/samples/spam', { gid, message }),

  addHam: (gid: string, message: string) =>
    apiClient.post('/samples/ham', { gid, message }),

  deleteSpam: (gid: string, message: string) =>
    apiClient.delete('/samples/spam', { data: { gid, message } }),

  deleteHam: (gid: string, message: string) =>
    apiClient.delete('/samples/ham', { data: { gid, message } }),

  reload: (gid: string) =>
    apiClient.put('/samples/reload', { gid }),
};

import axios from 'axios';

const apiClient = axios.create({
  baseURL: '/api/v2',
  headers: {
    'Content-Type': 'application/json',
  },
});

// request interceptor: attach access token
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// response interceptor: handle 401 with token refresh
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      const refreshToken = localStorage.getItem('refresh_token');
      if (!refreshToken) {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        return Promise.reject(error);
      }

      try {
        const response = await axios.post('/api/v2/auth/refresh', {
          refresh_token: refreshToken,
        });
        const { access_token, refresh_token } = response.data;

        localStorage.setItem('access_token', access_token);
        localStorage.setItem('refresh_token', refresh_token);

        originalRequest.headers.Authorization = `Bearer ${access_token}`;
        return apiClient(originalRequest);
      } catch {
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        return Promise.reject(error);
      }
    }

    return Promise.reject(error);
  }
);

export default apiClient;

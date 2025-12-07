import axios from 'axios';

// API Base URL: 환경 변수에서 가져오거나 상대 경로 사용
// 상대 경로를 사용하면 Ingress를 통해 자동으로 라우팅됨
const API_BASE_URL = process.env.REACT_APP_API_URL || '/api/v1';

// Create axios instance
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token to requests
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Handle auth errors
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Unauthorized - redirect to login
      localStorage.removeItem('auth_token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Auth API
export const authAPI = {
  login: async (username: string, password: string) => {
    const response = await apiClient.post('/auth/login', { username, password });
    // Log response for debugging
    console.log('Login response:', response.data);
    // Check both response.data.token and response.data.data.token
    const token = response.data.token || response.data.data?.token;
    if (token) {
      localStorage.setItem('auth_token', token);
      console.log('Token saved to localStorage');
    } else {
      console.error('No token in login response:', response.data);
    }
    return response.data;
  },
  logout: () => {
    localStorage.removeItem('auth_token');
  },
  isAuthenticated: () => {
    return !!localStorage.getItem('auth_token');
  },
};

// Resource API
export const resourceAPI = {
  // Instances
  listInstances: () => apiClient.get('/instances'),
  getInstance: (id: string) => apiClient.get(`/instances/${id}`),
  
  // Projects
  listProjects: () => apiClient.get('/projects'),
  getProject: (id: string) => apiClient.get(`/projects/${id}`),
  
  // Networks
  listNetworks: () => apiClient.get('/networks'),
  getNetwork: (id: string) => apiClient.get(`/networks/${id}`),
  
  // Hypervisors
  listHypervisors: () => apiClient.get('/hypervisors'),
  getHypervisor: (id: string) => apiClient.get(`/hypervisors/${id}`),
  
  // Flavors
  listFlavors: () => apiClient.get('/flavors'),
  getFlavor: (id: string) => apiClient.get(`/flavors/${id}`),
  
  // Volumes
  listVolumes: () => apiClient.get('/volumes'),
  getVolume: (id: string) => apiClient.get(`/volumes/${id}`),
};

// Topology API
export const topologyAPI = {
  getInstanceTopology: (id: string, maxDepth?: number) => {
    const params = maxDepth ? { max_depth: maxDepth } : {};
    return apiClient.get(`/topology/instances/${id}`, { params });
  },
  getHostTopology: (id: string, maxDepth?: number) => {
    const params = maxDepth ? { max_depth: maxDepth } : {};
    return apiClient.get(`/topology/hosts/${id}`, { params });
  },
  getNetworkTopology: (id: string) => apiClient.get(`/topology/networks/${id}`),
};

// Metrics API
export const metricsAPI = {
  getInstanceMetrics: (id: string, startTime?: string, endTime?: string) => {
    const params: any = {};
    if (startTime) params.start_time = startTime;
    if (endTime) params.end_time = endTime;
    return apiClient.get(`/metrics/instances/${id}`, { params });
  },
  getNetworkMetrics: (id: string, startTime?: string, endTime?: string) => {
    const params: any = {};
    if (startTime) params.start_time = startTime;
    if (endTime) params.end_time = endTime;
    return apiClient.get(`/metrics/networks/${id}`, { params });
  },
  getHypervisorMetrics: (id: string, startTime?: string, endTime?: string) => {
    const params: any = {};
    if (startTime) params.start_time = startTime;
    if (endTime) params.end_time = endTime;
    return apiClient.get(`/metrics/hypervisors/${id}`, { params });
  },
};

// Events API (SSE)
export const eventsAPI = {
  createEventSource: () => {
    // Note: EventSource doesn't support custom headers, so we'll need to pass token as query param
    // or use a different approach
    const token = localStorage.getItem('auth_token');
    // 상대 경로 사용: 현재 호스트를 기준으로 자동으로 경로 생성
    const baseUrl = API_BASE_URL.startsWith('http') 
      ? API_BASE_URL 
      : `${window.location.origin}${API_BASE_URL}`;
    const url = token 
      ? `${baseUrl}/events/stream?token=${token}`
      : `${baseUrl}/events/stream`;
    return new EventSource(url, {
      withCredentials: false,
    } as any);
  },
};

export default apiClient;


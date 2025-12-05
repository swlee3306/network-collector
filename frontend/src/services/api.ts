import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1';

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
    if (response.data.token) {
      localStorage.setItem('auth_token', response.data.token);
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
    const url = token 
      ? `${API_BASE_URL}/events/stream?token=${token}`
      : `${API_BASE_URL}/events/stream`;
    return new EventSource(url, {
      withCredentials: false,
    } as any);
  },
};

export default apiClient;


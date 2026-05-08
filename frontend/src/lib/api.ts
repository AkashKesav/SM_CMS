import { AccessUser, ApiResponse, ProvisionResult, StudentChangeRequest, StudentContributor, StudentSearchResult, StudentWorkspace } from '@/types/schema';
import { useAuthStore } from '@/stores/authStore';

const getBaseUrl = (): string => {
  // Use relative URL so Vite proxy works in dev, or explicit URL if configured
  return import.meta.env.VITE_BACKEND_URL || '';
};

const getAuthToken = (): string | null => {
  const state = useAuthStore.getState();
  return state.tokens?.access_token || null;
};

async function request<T = any>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const url = `${getBaseUrl()}/api${endpoint}`;
  const token = getAuthToken();

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...options.headers as Record<string, string>,
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const response = await fetch(url, {
      ...options,
      headers,
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error || `HTTP ${response.status}`);
    }

    return data as ApiResponse<T>;
  } catch (error: any) {
    return {
      success: false,
      error: error.message || 'Request failed',
    };
  }
}

// Auth API
export const authApi = {
  login: async (email: string, password: string) => {
    return request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  },

  register: async (email: string, password: string, name?: string, rollNo?: string) => {
    return request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, password, name, roll_no: rollNo }),
    });
  },

  logout: async () => {
    return request('/auth/logout', {
      method: 'POST',
    });
  },

  refreshToken: async (refreshToken: string) => {
    return request('/auth/refresh', {
      method: 'POST',
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
  },

  me: async () => {
    return request('/me');
  },

  changePassword: async (password: string) => {
    return request('/password', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  },
};

export const studentApi = {
  me: async () => {
    return request<StudentWorkspace>('/student/me');
  },

  listRequests: async () => {
    return request<StudentChangeRequest[]>('/student/requests');
  },

  createRequest: async (data: Partial<StudentChangeRequest>) => {
    return request<StudentChangeRequest>('/student/requests', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  searchStudents: async (query: string, limit = 20) => {
    const params = new URLSearchParams({ q: query, limit: String(limit) });
    return request<StudentSearchResult[]>(`/student/search?${params.toString()}`);
  },

  requestAchievementContributor: async (payload: { achievement_id: string; student_id?: string }) => {
    return request<StudentChangeRequest>('/student/achievements/contributors', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  requestProjectContributor: async (payload: { project_id: string; student_id?: string; role?: string }) => {
    return request<StudentChangeRequest>('/student/projects/contributors', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  getContributors: async (type: 'achievement' | 'project', id: string) => {
    const params = new URLSearchParams({ type, id });
    return request<StudentContributor[]>(`/student/contributors?${params.toString()}`);
  },
};

export const adminApi = {
  listChangeRequests: async (status = 'pending') => {
    return request<StudentChangeRequest[]>(`/admin/change-requests?status=${encodeURIComponent(status)}`);
  },

  approveChangeRequest: async (id: string, note = '') => {
    return request<StudentChangeRequest>(`/admin/change-requests/${encodeURIComponent(id)}/approve`, {
      method: 'POST',
      body: JSON.stringify({ note }),
    });
  },

  rejectChangeRequest: async (id: string, note = '') => {
    return request<StudentChangeRequest>(`/admin/change-requests/${encodeURIComponent(id)}/reject`, {
      method: 'POST',
      body: JSON.stringify({ note }),
    });
  },

  listUsers: async () => {
    return request<AccessUser[]>('/admin/users');
  },

  provisionStudents: async (password = 'user123') => {
    return request<ProvisionResult>('/admin/users/provision', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  },

  setUserRole: async (userId: string, role: 'admin' | 'student') => {
    return request(`/admin/users/${encodeURIComponent(userId)}/role`, {
      method: 'POST',
      body: JSON.stringify({ role }),
    });
  },

  resetUserPassword: async (userId: string, password = 'user123') => {
    return request(`/admin/users/${encodeURIComponent(userId)}/password`, {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  },
};

// Schema API
export const schemaApi = {
  getSchema: async () => {
    return request('/schema');
  },

  refreshSchema: async () => {
    return request('/schema/refresh', {
      method: 'POST',
    });
  },
};

// Records API (generic CRUD)
export const recordsApi = {
  list: async (
    tableName: string,
    params: Record<string, any> = {}
  ) => {
    const queryParams = new URLSearchParams();
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        queryParams.set(key, String(value));
      }
    });

    const query = queryParams.toString();
    return request(`/tables/${tableName}${query ? `?${query}` : ''}`);
  },

  get: async (tableName: string, id: string) => {
    return request(`/tables/${tableName}/${encodeURIComponent(id)}`);
  },

  create: async (tableName: string, data: Record<string, any>) => {
    return request(`/tables/${tableName}`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  update: async (tableName: string, id: string, data: Record<string, any>) => {
    return request(`/tables/${tableName}/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  delete: async (tableName: string, id: string, soft = true) => {
    return request(`/tables/${tableName}/${encodeURIComponent(id)}?soft=${soft}`, {
      method: 'DELETE',
    });
  },
};

// Storage API (Supabase direct upload)
export const storageApi = {
  getSignedUrl: async (bucket: string, fileName: string, contentType: string) => {
    return request('/upload/sign', {
      method: 'POST',
      body: JSON.stringify({
        bucket,
        file_name: fileName,
        content_type: contentType,
      }),
    });
  },

  uploadToSupabase: async (signedUrl: string, file: File): Promise<string> => {
    const response = await fetch(signedUrl, {
      method: 'PUT',
      body: file,
      headers: {
        'Content-Type': file.type,
      },
    });

    if (!response.ok) {
      throw new Error('Upload failed');
    }

    // Convert the signed upload URL to the public URL
    // Signed URL: https://xxx.supabase.co/storage/v1/object/upload/sign/{bucket}/{path}?token=...
    // Public URL: https://xxx.supabase.co/storage/v1/object/public/{bucket}/{path}
    const urlWithoutQuery = signedUrl.split('?')[0];
    const publicUrl = urlWithoutQuery.replace('/object/upload/sign/', '/object/public/');
    return publicUrl;
  },
};

export default request;

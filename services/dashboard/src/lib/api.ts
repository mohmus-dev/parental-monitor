import type {
  Parent,
  Child,
  LoginResponse,
  RegisterResponse,
  DeleteChildrenResponse,
  PairingInvite,
  DashboardSummary,
  DashboardAlert,
  DashboardActivity,
  PaginatedAlertResponse,
  PaginatedSearchResponse,
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '';

function getCookie(name: string): string {
  const value = document.cookie
    .split('; ')
    .find((entry) => entry.startsWith(`${name}=`));
  if (!value) return '';
  return decodeURIComponent(value.split('=').slice(1).join('='));
}

async function ensureCSRFToken(): Promise<string> {
  const existing = getCookie('csrf_token');
  if (existing) return existing;

  const response = await fetch(`${API_BASE_URL}/api/v1/auth/csrf`, {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  });
  if (!response.ok) {
    throw new Error('Unable to initialize secure session');
  }
  const data = await response.json() as { csrf_token?: string };
  return data.csrf_token ?? '';
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const method = (init.method ?? 'GET').toUpperCase();
  const headers = new Headers(init.headers ?? {});
  if (method !== 'GET' && method !== 'HEAD' && !path.startsWith('/api/v1/devices/pair')) {
    const csrfToken = await ensureCSRFToken();
    if (csrfToken) headers.set('X-CSRF-Token', csrfToken);
    if (init.body) headers.set('Content-Type', 'application/json');
  } else if (init.body) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: 'include',
    headers,
  });

  if (!response.ok) {
    // Some handlers return JSON errors, others plain text — handle both.
    const text = await response.text();
    let message = text || `Request failed (${response.status})`;
    try {
      const parsed = JSON.parse(text);
      if (parsed.message) message = parsed.message;
      if (parsed.error === 'session_expired') {
        // Session expired — force re-login
        window.dispatchEvent(new Event('session:expired'));
      }
    } catch {
      /* plain text error, keep as-is */
    }
    throw new Error(message);
  }

  return (await response.json()) as T;
}

// ---- Auth ----
export const registerParent = (body: {
  name: string;
  email: string;
  password: string;
  app_name?: string;
  platform?: string;
  device_id?: string;
}) => api<RegisterResponse>('/api/v1/parents/register', {
  method: 'POST',
  body: JSON.stringify(body),
});

export const login = (email: string, password: string) =>
  api<LoginResponse>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });

export const logout = () => api<{ message?: string }>('/api/v1/auth/logout', {
  method: 'POST',
});

// ---- Parent ----
export const getCurrentParent = () => api<Parent>('/api/v1/parents/me');

export const deleteAccount = () => api<unknown>('/api/v1/parents/me', {
  method: 'DELETE',
});

// ---- Children ----
export const getChildren = () => api<Child[]>('/api/v1/parents/me/children');

export const getDashboardSummary = () =>
  api<DashboardSummary>('/api/v1/parents/me/dashboard');

export const getParentAlerts = (page = 1, limit = 20) =>
  api<PaginatedAlertResponse>(`/api/v1/parents/me/alerts?page=${page}&limit=${limit}`);

export const getParentSearches = (page = 1, limit = 20) =>
  api<PaginatedSearchResponse>(`/api/v1/parents/me/searches?page=${page}&limit=${limit}`);

export const registerChild = (body: {
  name: string;
  age?: number;
  device_id?: string;
  app_name?: string;
  platform?: string;
}) => api<Child>('/api/v1/parents/me/children', {
  method: 'POST',
  body: JSON.stringify(body),
});

export const createPairingInvite = (childId: string) =>
  api<PairingInvite>(`/api/v1/parents/me/children/${encodeURIComponent(childId)}/pairing-invites`, {
    method: 'POST',
  });

export const deleteChildren = (childIds: string[]) =>
  api<DeleteChildrenResponse>('/api/v1/parents/me/children', {
    method: 'DELETE',
    body: JSON.stringify({ child_ids: childIds }),
  });
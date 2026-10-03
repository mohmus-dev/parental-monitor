export type Parent = {
  id: string;
  name: string;
  email: string;
  created_at: string;
  app_name?: string;
  platform?: string;
  device_id?: string;
};

export type Child = {
  id: string;
  parent_id: string;
  name: string;
  age?: number;
  device_id?: string;
  app_name?: string;
  platform?: string;
  created_at: string;
};

export type PairingInvite = {
  code: string;
  expires_at: string;
};

export type DashboardActivity = {
  id: string;
  child_id: string;
  child_name: string;
  query: string;
  engine: string;
  timestamp: string;
};

export type DashboardAlert = {
  id: string;
  child_id: string;
  device_id?: string;
  category: string;
  keyword: string;
  query: string;
  timestamp: string;
};

export type DashboardSummary = {
  total_children: number;
  paired_devices: number;
  pending_pairing: number;
  alerts_count: number;
  recent_activity: DashboardActivity[];
  alerts: DashboardAlert[];
};

export type PaginatedAlertResponse = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  items: DashboardAlert[];
};

export type PaginatedSearchResponse = {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
  items: DashboardActivity[];
};

export type LoginResponse = {
  token: string;
  expiry: string;
};

export type RegisterResponse = {
  parent: Parent;
  message: string;
};

export type DeleteChildrenResponse = {
  message: string;
};
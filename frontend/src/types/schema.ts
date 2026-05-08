// Core schema types discovered at runtime

export interface ColumnInfo {
  column_name: string;
  data_type: string;
  is_nullable: string;
  column_default: string | null;
  character_maximum_length: number | null;
  numeric_precision: number | null;
  udt_name: string;
  is_primary_key: boolean;
  is_foreign_key: boolean;
}

export interface ForeignKeyInfo {
  table_name: string;
  column_name: string;
  foreign_table_name: string;
  foreign_column_name: string;
  constraint_name: string;
}

export interface TableSchema {
  table_name: string;
  columns: ColumnInfo[];
  foreign_keys: ForeignKeyInfo[];
  primary_key: string;
  primary_keys?: string[];
  has_timestamps: boolean;
}

export interface DatabaseSchema {
  tables: TableSchema[];
  discovered_at: string;
  version: number;
}

// UI Field types mapped from PostgreSQL types
export type FieldType = 
  | 'text'
  | 'number'
  | 'boolean'
  | 'date'
  | 'datetime'
  | 'json'
  | 'uuid'
  | 'email'
  | 'url'
  | 'textarea'
  | 'image'
  | 'select'
  | 'multiselect'
  | 'code';

export interface FieldConfig {
  name: string;
  label: string;
  type: FieldType;
  required: boolean;
  nullable: boolean;
  defaultValue?: any;
  maxLength?: number;
  precision?: number;
  foreignKey?: ForeignKeyInfo;
  options?: { label: string; value: any }[];
  placeholder?: string;
  description?: string;
}

// API Response types
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: string;
  details?: string;
  message?: string;
  meta?: PaginatedMeta;
}

export interface PaginatedMeta {
  data: Record<string, any>[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// Auth types
export interface User {
  id: string;
  email: string;
  role: 'admin' | 'student' | 'editor' | 'viewer' | 'authenticated';
  name?: string;
  avatar_url?: string;
  student_id?: string;
}

export interface AuthTokens {
  access_token: string;
  refresh_token: string;
  user: User;
}

export interface StudentChangeRequest {
  id: string;
  requester_user_id: string;
  requester_name?: string;
  requester_roll_no?: string;
  student_id?: string;
  request_type: 'claim_student' | 'create' | 'update' | 'delete' | 'link' | 'unlink';
  target_table: string;
  target_key?: Record<string, any>;
  current_data?: Record<string, any>;
  proposed_data: Record<string, any>;
  status: 'pending' | 'approved' | 'rejected';
  admin_note?: string;
  reviewed_by?: string;
  reviewed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface StudentWorkspace {
  profile: User;
  student?: Record<string, any>;
  linked_records: Record<string, Record<string, any>[]>;
  references: Record<string, Record<string, any>[]>;
  requests: StudentChangeRequest[];
}

export interface StudentSearchResult {
  id: string;
  name: string;
  roll_no: string;
  image_url?: string | null;
  batch_id?: string | null;
}

export interface StudentContributor {
  id: string;
  name: string;
  roll_no: string;
  image_url?: string | null;
  role?: string | null;
}

export interface AccessUser {
  user_id?: string;
  student_id: string;
  roll_no: string;
  name: string;
  login_email: string;
  role: 'admin' | 'student';
  has_account: boolean;
}

export interface ProvisionUserResult {
  student_id: string;
  roll_no: string;
  login_email: string;
  role: 'admin' | 'student';
  status: 'created' | 'updated' | 'failed';
  message?: string;
  user_id?: string;
}

export interface ProvisionResult {
  created: number;
  updated: number;
  failed: number;
  users: ProvisionUserResult[];
}

// Table view config
export interface TableFilter {
  column: string;
  operator: 'eq' | 'ne' | 'gt' | 'lt' | 'like';
  value: string;
}

export interface TableSort {
  column: string;
  direction: 'ASC' | 'DESC';
}

export interface TableViewConfig {
  pageSize: number;
  sort: TableSort;
  filters: TableFilter[];
  visibleColumns: string[];
}

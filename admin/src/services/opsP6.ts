import { request } from '@umijs/max';

export type BackupJob = {
  id: string;
  backupId: string;
  backupType: string;
  environment: string;
  status: string;
  verificationStatus: string;
  storageProvider: string;
  encrypted: boolean;
  artifactSize?: number;
  createdAt: string;
  completedAt?: string;
  errorSummary?: string;
};

export type RestoreJob = {
  id: string;
  restoreId: string;
  backupId: string;
  targetEnvironment: string;
  status: string;
  safetyGateStatus: string;
  validationStatus?: string;
  createdAt: string;
  completedAt?: string;
  errorSummary?: string;
};

type ListResult<T> = {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
};

export async function fetchBackups(params?: { page?: number; pageSize?: number }) {
  return request<{ data: ListResult<BackupJob> }>('/api/v1/ops/backups', { method: 'GET', params });
}

export async function createBackup(data?: { dryRun?: boolean; reason?: string }) {
  return request<{ data: BackupJob }>('/api/v1/ops/backups', { method: 'POST', data });
}

export async function verifyBackup(id: string) {
  return request(`/api/v1/ops/backups/${id}/verify`, { method: 'POST' });
}

export async function holdBackup(id: string, reason: string) {
  return request(`/api/v1/ops/backups/${id}/hold`, {
    method: 'POST',
    data: { holdType: 'manual_hold', reason },
  });
}

export async function fetchRestores(params?: { page?: number; pageSize?: number }) {
  return request<{ data: ListResult<RestoreJob> }>('/api/v1/ops/restores', { method: 'GET', params });
}

export async function createRestore(data: {
  backupId: string;
  targetEnvironment: string;
  targetDatabaseName: string;
  targetIsIsolated: boolean;
  operatorReauthenticated: boolean;
  highRiskConfirmed: boolean;
}) {
  return request<{ data: RestoreJob }>('/api/v1/ops/restores', { method: 'POST', data });
}

export async function verifyRestore(id: string) {
  return request(`/api/v1/ops/restores/${id}/verify`, { method: 'POST' });
}

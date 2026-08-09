import { getJSON, postJSON, putJSON } from '@/services/request';

export type P10RuntimeControl = {
  tenantId: number;
  providerKillActive: boolean;
  tenantKillActive: boolean;
  shopKillActive: boolean;
  readKillActive: boolean;
  writeKillActive: boolean;
  revision: number;
};

export type P10Allowlist = {
  tenantId: number;
  shopId: string;
  enabled: boolean;
  revision: number;
};

export type P10GrayPolicy = {
  tenantId: number;
  shopId: string;
  maxSku: number;
  status: 'draft' | 'pending_approval' | 'approved' | 'active' | 'paused' | 'stopped';
  ownerApproved: boolean;
  technicalLeadApproved: boolean;
  revision: number;
};

export type P10LastRead = {
  runId: string;
  requestId?: string;
  providerRequestId?: string;
  status: string;
  providerMode: string;
  revision: number;
  lastErrorCode?: string;
  rateLimited: boolean;
  retryAfterSeconds?: number;
  startedAt?: string;
  finishedAt?: string;
};

export type P10RuntimeStatus = {
  currentAllowedLevel: 'L0';
  environment: string;
  developmentStatus: string;
  verificationStatus: string;
  manualAcceptanceStatus: string;
  externalActivationStatus: string;
  providerProtocolMappingStatus: string;
  realProviderEnabled: boolean;
  realPlatformNetworkEnabled: boolean;
  realCredentialsEnabled: boolean;
  realInventoryReadEnabled: boolean;
  realInventoryWriteEnabled: false;
  inventoryMutationEnabled: false;
  backgroundWorkerEnabled: false;
  automaticRetryEnabled: false;
  readOnlyCapability: boolean;
  offlineOAuthEnabled: boolean;
  offlineCredentialAvailable: boolean;
  control: P10RuntimeControl;
  allowlist?: P10Allowlist;
  gray?: P10GrayPolicy;
  lastRead?: P10LastRead;
  initialLimits: { maxTenant: number; maxShop: number; maxSku: number };
  productionReady: false;
  productionAcceptancePassed: false;
};

export type P10CredentialMetadata = {
  credentialId: string;
  tenantId: number;
  platform: string;
  shopId: string;
  status: string;
  expiresAt?: string;
  rotatedAt?: string;
  revokedAt?: string;
  createdAt: string;
  updatedAt: string;
  version: number;
  keyReference?: string;
  algorithm?: string;
};

export type P10ReadRunResult = {
  runId: string;
  status: string;
  snapshotCount: number;
  calibrationCount: number;
  manualReviewCount: number;
  requestId: string;
  providerMode: string;
  automaticRetryUsed: false;
};

const BASE = '/api/v1/p10';

export function createP10IdempotencyKey(action: string) {
  const random = Math.random().toString(36).slice(2, 12);
  return `p10-ui:${action}:${Date.now()}:${random}`;
}

export function getP10Status() {
  return getJSON<P10RuntimeStatus>(`${BASE}/status`);
}

export async function listP10Credentials() {
  const result = await getJSON<{ items: P10CredentialMetadata[] }>(`${BASE}/credentials`);
  return result.items ?? [];
}

export function createOfflineP10Credential(shopId: string) {
  return postJSON<P10CredentialMetadata>(`${BASE}/credentials/offline`, { platform: 'douyin', shopId });
}

export function rotateP10Credential(credentialId: string, expectedRevision: number) {
  return postJSON<P10CredentialMetadata>(`${BASE}/credentials/${encodeURIComponent(credentialId)}/rotate`, {
    expectedRevision,
  });
}

export function revokeP10Credential(credentialId: string, expectedRevision: number) {
  return postJSON<P10CredentialMetadata>(`${BASE}/credentials/${encodeURIComponent(credentialId)}/revoke`, {
    expectedRevision,
  });
}

export function startOfflineP10OAuth(shopId: string, redirectUri: string) {
  return postJSON<{ authorizationUrl: string; mode: 'offline_fixture'; networkRequestExecuted: false }>(
    `${BASE}/credentials/oauth/offline/start`,
    { platform: 'douyin', shopId, redirectUri },
  );
}

export function completeOfflineP10OAuth(state: string) {
  return postJSON<P10CredentialMetadata>(`${BASE}/credentials/oauth/offline/complete`, { state });
}

export function updateP10KillSwitches(body: {
  providerKillActive: boolean;
  tenantKillActive: boolean;
  shopKillActive: boolean;
  readKillActive: boolean;
  expectedRevision: number;
}) {
  return putJSON<P10RuntimeControl, typeof body>(`${BASE}/controls/kill-switches`, body);
}

export function updateP10Allowlist(body: { shopId: string; enabled: boolean; expectedRevision: number }) {
  return putJSON<P10Allowlist, typeof body>(`${BASE}/controls/allowlist`, body);
}

export function saveP10GrayDraft(body: { shopId: string; maxSku: number; expectedRevision: number }) {
  return putJSON<P10GrayPolicy, typeof body>(`${BASE}/gray`, body);
}

export function pauseP10Gray(expectedRevision: number) {
  return postJSON<P10GrayPolicy>(`${BASE}/gray/pause`, { expectedRevision });
}

export function stopP10Gray(expectedRevision: number) {
  return postJSON<P10GrayPolicy>(`${BASE}/gray/stop`, { expectedRevision });
}

export function createP10InventoryReadRun(shopId: string, idempotencyKey: string) {
  return postJSON<P10ReadRunResult>(`${BASE}/inventory-read/runs`, { shopId }, {
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}

export function rerunP10InventoryRead(runId: string, expectedRevision: number, idempotencyKey: string) {
  return postJSON<P10ReadRunResult>(
    `${BASE}/inventory-read/runs/${encodeURIComponent(runId)}/rerun`,
    { expectedRevision },
    { headers: { 'Idempotency-Key': idempotencyKey } },
  );
}

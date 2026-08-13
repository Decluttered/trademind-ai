import { ok } from './envelope';

export const E2E_OPERATION_TASK_ID = 'e2e-operation-task-1';

export const e2eOperationTask = {
  id: E2E_OPERATION_TASK_ID,
  sourceType: 'manual',
  sourceReference: 'e2e-product-1',
  taskType: 'product_content',
  platform: 'local',
  title: 'E2E 商品内容复核',
  summary: '检查标题、卖点和规格信息',
  status: 'execution_failed',
  priority: 'high',
  revision: 7,
  latestDraftVersion: 2,
  latestExecutionStatus: 'failed',
  createdBy: 'e2e-user',
  createdAt: '2026-08-05T09:00:00.000Z',
  updatedAt: '2026-08-05T10:00:00.000Z',
};

export const e2eOperationDraft = {
  draftId: 'e2e-operation-draft-2',
  draftVersion: 2,
  adapterMode: 'local_draft_only',
  payloadHash: 'e2e-safe-payload-hash',
  status: 'approved',
  changeReason: '补充商品卖点',
  createdBy: 'e2e-user',
  createdAt: '2026-08-05T09:10:00.000Z',
  updatedAt: '2026-08-05T09:20:00.000Z',
};

export const e2eProductionStatus = {
  currentAllowedLevel: 'L0',
  environment: 'test',
  realProviderEnabled: false,
  realPlatformNetworkEnabled: false,
  realCredentialsEnabled: false,
  realProductDraftWriteEnabled: false,
  backgroundWorkerEnabled: false,
  productPublishQueueEnabled: false,
  providerWriteReady: false,
  automaticRetryEnabled: false,
  control: {
    providerKillActive: true,
    tenantKillActive: true,
    shopKillActive: true,
    readKillActive: true,
    writeKillActive: true,
    revision: 1,
  },
  initialLimits: { maxTenant: 1, maxShop: 1, maxSku: 100 },
  productionReady: false,
  productionAcceptancePassed: false,
};

export const e2eOperationAttempt = {
  attemptId: 'e2e-operation-attempt-1',
  attemptNumber: 1,
  status: 'failed',
  adapterMode: 'local_draft_only',
  platform: 'local',
  approvedDraftVersion: 2,
  approvedDraftPayloadHash: e2eOperationDraft.payloadHash,
  executedDraftVersion: 2,
  executedDraftPayloadHash: e2eOperationDraft.payloadHash,
  resultType: '',
  requestId: 'e2e-operation-request-1',
  startedAt: '2026-08-05T09:30:00.000Z',
  finishedAt: '2026-08-05T09:31:00.000Z',
  createdAt: '2026-08-05T09:30:00.000Z',
};

export const e2eOperationTaskDetail = {
  ...e2eOperationTask,
  payload: { title: 'E2E 待复核标题', sellingPoints: ['稳定测试数据'] },
  latestDraft: e2eOperationDraft,
  latestAttempt: e2eOperationAttempt,
  allowedActions: {
    canEditDraft: true,
    canApprove: false,
    canReject: false,
    canExecute: false,
    canRetry: true,
    canCancel: true,
  },
};

export const e2eOperationEvent = {
  eventId: 'e2e-operation-event-1',
  sequence: 1,
  eventType: 'execution_failed',
  actorType: 'user',
  actorId: 'e2e-user',
  beforeState: 'executing',
  afterState: 'execution_failed',
  platformDraftId: e2eOperationDraft.draftId,
  draftVersion: 2,
  requestId: 'e2e-operation-request-1',
  reason: '测试环境中的安全失败记录',
  metadata: { retryable: true, token: 'must-not-render' },
  occurredAt: '2026-08-05T09:31:00.000Z',
};

export function operationTaskResponse(path: string) {
  if (path === '/api/v1/p10/status') return ok(e2eProductionStatus);
  if (path === '/api/v1/operation-tasks') {
    return ok({ items: [e2eOperationTask], nextCursor: '', hasMore: false, limit: 20 });
  }
  if (path === `/api/v1/operation-tasks/${E2E_OPERATION_TASK_ID}`) return ok(e2eOperationTaskDetail);
  if (path === `/api/v1/operation-tasks/${E2E_OPERATION_TASK_ID}/drafts`) {
    return ok({ items: [e2eOperationDraft], limit: 50 });
  }
  if (path === `/api/v1/operation-tasks/${E2E_OPERATION_TASK_ID}/attempts`) {
    return ok({ items: [e2eOperationAttempt], nextCursor: '', hasMore: false, limit: 20 });
  }
  if (path === `/api/v1/operation-tasks/${E2E_OPERATION_TASK_ID}/events`) {
    return ok({ items: [e2eOperationEvent], nextSequence: 0, hasMore: false, limit: 30 });
  }
  return null;
}

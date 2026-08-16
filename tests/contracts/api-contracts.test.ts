import { describe, expect, it } from 'vitest';
import contracts from './api-contracts.json';

const routeKey = (endpoint: { method: string; path: string }) => `${endpoint.method} ${endpoint.path}`;

describe('TradeMind API contract registry', () => {
  it('keeps the backend envelope explicit for frontend and E2E mocks', () => {
    expect(contracts.envelope.success).toEqual(['code', 'message', 'data']);
    expect(contracts.envelope.optional).toContain('traceId');
    expect(contracts.envelope.errorCodeRule).toContain('non-zero');
  });

  it('covers core Admin production endpoints', () => {
    const routes = new Set(contracts.endpoints.map(routeKey));

    expect(routes).toEqual(
      new Set([
        'GET /api/v1/auth/profile',
        'GET /api/v1/image/providers',
        'GET /api/v1/p10/status',
        'POST /api/v1/operation-tasks',
        'GET /api/v1/operation-tasks/:id',
        'POST /api/v1/operation-tasks/:id/approve',
        'POST /api/v1/operation-tasks/:id/execute',
        'GET /api/v1/observability/overview',
        'GET /api/v1/observability/alerts',
        'POST /api/v1/observability/alerts/:id/ack',
        'POST /api/v1/observability/alerts/:id/silence',
        'GET /api/v1/products/:id',
        'GET /api/v1/products/:id/readiness',
        'GET /api/v1/products/:id/publications',
        'GET /api/v1/product-publications/:id/douyin/sku-bindings',
        'GET /api/v1/products/:id/publish-targets',
        'POST /api/v1/products/:id/publish-targets/create-drafts',
        'POST /api/v1/product-publish/batch-targets/create-drafts',
        'POST /api/v1/product-publish/tasks/:id/retry',
        'POST /api/v1/product-publish/tasks/:id/recover-douyin-draft',
        'POST /api/v1/product-publish/batches/:id/retry-failed',
        'POST /api/v1/products/:id/platform-configs/douyin_shop/create-draft',
        'POST /api/v1/products/:id/publish',
        'GET /api/v1/customer/dashboard',
        'GET /api/v1/customer/auto-reply-setting',
        'PUT /api/v1/customer/auto-reply-setting',
        'GET /api/v1/customer/shops/:shopId/auto-reply-policy',
        'PUT /api/v1/customer/shops/:shopId/auto-reply-policy',
        'GET /api/v1/customer/shops/:shopId/auto-reply-runs',
        'POST /api/v1/customer/conversations/:id/send-platform-message',
        'POST /v1/discovery-runs',
        'GET /v1/products',
        'GET /v1/collections',
        'POST /v1/collections',
        'POST /v1/collections/:id/products',
        'POST /v1/listing-drafts',
        'GET /v1/listing-drafts',
        'GET /v1/listing-drafts/:id',
        'POST /v1/listing-drafts/:id/validate',
        'POST /v1/listing-drafts/:id/generate',
        'POST /v1/image-assets',
        'GET /v1/gpsr-profiles',
        'POST /v1/gpsr-profiles',
        'POST /v1/extension-tokens',
        'DELETE /v1/extension-tokens/:id',
        'POST /v1/extension/captures',
        'GET /api/v1/shops/:id/oauth/ebay/authorize-url',
        'POST /api/v1/shops/:id/oauth/ebay/callback',
        'GET /api/v1/platform/ebay/categories/:categoryId/aspects',
        'POST /api/v1/platform/ebay/categories/:categoryId/aspects/sync',
        'GET /v1/calendar/slots',
        'POST /v1/calendar/preview',
        'POST /v1/calendar/apply',
        'POST /v1/publications/:id/approve',
        'POST /internal/v1/mindbay/publications/:id/revalidate',
        'POST /internal/v1/mindbay/publications/:id/publish',
        'POST /internal/v1/mindbay/publications/:id/reconcile',
        'GET /v1/monitorable-listings',
        'POST /v1/monitor-runs',
        'GET /v1/price-rules',
        'POST /v1/price-rules',
        'GET /v1/price-decisions',
        'POST /v1/price-decisions/:id/apply',
        'GET /v1/profit/report',
      ]),
    );
  });

  it('defines payload/query contracts for state-changing publish APIs', () => {
    const createDraft = contracts.endpoints.find((item) => routeKey(item) === 'POST /api/v1/products/:id/platform-configs/douyin_shop/create-draft');
    const publish = contracts.endpoints.find((item) => routeKey(item) === 'POST /api/v1/products/:id/publish');
    const readiness = contracts.endpoints.find((item) => routeKey(item) === 'GET /api/v1/products/:id/readiness');

    expect(createDraft?.requestBody).toEqual(['shopId', 'publishMode', 'force']);
    expect(publish?.requestBody).toEqual(['shopId', 'options', 'force']);
    expect(readiness?.query).toEqual(['platform', 'shopId', 'mode']);
  });

  it('keeps Douyin writes exclusive to approved operation tasks', () => {
    const endpoint = (key: string) => contracts.endpoints.find((item) => routeKey(item) === key) as {
      fixedError?: { httpStatus: number; dataErrorCode: string };
      douyinPolicy?: string;
    } | undefined;
    expect(endpoint('POST /api/v1/products/:id/platform-configs/douyin_shop/create-draft')?.fixedError).toEqual({
      httpStatus: 409,
      dataErrorCode: 'DOUYIN_OPERATION_TASK_REQUIRED',
    });
    expect(endpoint('POST /api/v1/products/:id/publish')?.douyinPolicy).toBe('reject_before_task_write');
    expect(endpoint('POST /api/v1/products/:id/publish-targets/create-drafts')?.douyinPolicy).toBe('reject_entire_request_before_write');
    expect(endpoint('POST /api/v1/product-publish/batch-targets/create-drafts')?.douyinPolicy).toBe('reject_entire_request_before_idempotency_or_batch_write');
    expect(endpoint('POST /api/v1/product-publish/tasks/:id/retry')?.douyinPolicy).toBe('reject_without_task_state_change');
    expect(endpoint('POST /api/v1/product-publish/batches/:id/retry-failed')?.douyinPolicy).toBe('reject_entire_batch_without_task_state_change');
  });

  it('requires fail-closed customer auto-reply and idempotent send fields', () => {
    const setting = contracts.endpoints.find((item) => routeKey(item) === 'PUT /api/v1/customer/auto-reply-setting');
    const policy = contracts.endpoints.find((item) => routeKey(item) === 'PUT /api/v1/customer/shops/:shopId/auto-reply-policy');
    const send = contracts.endpoints.find((item) => routeKey(item) === 'POST /api/v1/customer/conversations/:id/send-platform-message');

    expect(setting?.requestBody).toEqual(['messageSyncEnabled', 'autoReplyEnabled', 'pollIntervalSeconds']);
    expect(policy?.requestBody).toEqual([
      'enabled',
      'tone',
      'shopPolicy',
      'maxReplyRunes',
      'maxRepliesPerHour',
      'requireOrderContext',
      'lowRiskOnly',
    ]);
    expect(send?.requestBody).toEqual(['reply', 'clientMessageId', 'suggestionId']);
  });

  it('defines the reviewed production draft operation task contract', () => {
    const runtimeStatus = contracts.endpoints.find((item) => routeKey(item) === 'GET /api/v1/p10/status');
    const create = contracts.endpoints.find((item) => routeKey(item) === 'POST /api/v1/operation-tasks');
    const approve = contracts.endpoints.find((item) => routeKey(item) === 'POST /api/v1/operation-tasks/:id/approve');
    const execute = contracts.endpoints.find((item) => routeKey(item) === 'POST /api/v1/operation-tasks/:id/execute');

    expect(runtimeStatus?.requiredResponseFields).toEqual(['providerWriteReady', 'productionReady']);
    expect(create?.requestBody).toEqual(['sourceType', 'sourceReference', 'taskType', 'platform', 'title', 'summary', 'payload', 'priority']);
    expect(approve?.requestBody).toEqual(['draftVersion', 'draftPayloadHash', 'reason', 'comment', 'expectedTaskRevision']);
    expect(execute?.requestBody).toEqual(['expectedTaskRevision', 'adapterMode']);
  });

  it('limits manual Douyin reconciliation to unknown results', () => {
    const recover = contracts.endpoints.find(
      (item) => routeKey(item) === 'POST /api/v1/product-publish/tasks/:id/recover-douyin-draft',
    ) as {
      requestBody?: string[];
      requiredPermission?: string;
      douyinPolicy?: string;
      fixedStateError?: { httpStatus: number; dataErrorCode: string };
    } | undefined;

    expect(recover?.requestBody).toEqual([]);
    expect(recover?.requiredPermission).toBe('operationtask.execute');
    expect(recover?.douyinPolicy).toBe('read_only_reconcile_result_unknown_only');
    expect(recover?.fixedStateError).toEqual({
      httpStatus: 409,
      dataErrorCode: 'DOUYIN_RECOVERY_NOT_ALLOWED',
    });
  });

  it('defines filtered system alert queries and audited silence fields', () => {
    const overview = contracts.endpoints.find(
      (item) => routeKey(item) === 'GET /api/v1/observability/overview',
    );
    const list = contracts.endpoints.find(
      (item) => routeKey(item) === 'GET /api/v1/observability/alerts',
    );
    const acknowledge = contracts.endpoints.find(
      (item) => routeKey(item) === 'POST /api/v1/observability/alerts/:id/ack',
    );
    const silence = contracts.endpoints.find(
      (item) => routeKey(item) === 'POST /api/v1/observability/alerts/:id/silence',
    );

    expect(overview?.requiredResponseFields).toEqual([
      'overallStatus',
      'metrics',
      'alerts',
      'evaluation',
      'slo',
      'telemetry',
      'environment',
      'timestamp',
    ]);

    expect(list?.query).toEqual(['page', 'pageSize', 'status', 'severity', 'module']);
    expect(acknowledge?.requestBody).toEqual([]);
    expect(silence?.requestBody).toEqual(['reason', 'durationHours']);
  });

  it('marks every protected Admin endpoint as authenticated', () => {
    expect(contracts.endpoints).toHaveLength(64);
    expect(contracts.endpoints.every((endpoint) => endpoint.auth === true || endpoint.auth === 'mindbay-extension:capture' || endpoint.auth === 'temporal-service')).toBe(true);
  });

  it('keeps MindBay commands idempotent and extension auth separate',()=>{
    const phase1=contracts.endpoints.filter((endpoint)=>endpoint.path.startsWith('/v1/'));
    expect(phase1.filter((endpoint)=>endpoint.method==='POST'&&!endpoint.path.endsWith('extension-tokens')&&!endpoint.path.endsWith('/preview')).every((endpoint)=>endpoint.idempotencyKey===true)).toBe(true);
    expect(phase1.find((endpoint)=>endpoint.path==='/v1/extension/captures')?.auth).toBe('mindbay-extension:capture');
  });

  it('keeps calendar preview pure and calendar apply idempotent',()=>{
    const preview=contracts.endpoints.find((endpoint)=>routeKey(endpoint)==='POST /v1/calendar/preview');
    const apply=contracts.endpoints.find((endpoint)=>routeKey(endpoint)==='POST /v1/calendar/apply');
    expect(preview?.idempotencyKey).toBeUndefined();
    expect(apply?.idempotencyKey).toBe(true);
    expect(apply?.requestBody).toEqual(['shopId','marketplace','slots','publishConfig']);
  });

  it('keeps MindBay repricing decisions versioned and apply idempotent',()=>{
    const run=contracts.endpoints.find((endpoint)=>routeKey(endpoint)==='POST /v1/monitor-runs');
    const apply=contracts.endpoints.find((endpoint)=>routeKey(endpoint)==='POST /v1/price-decisions/:id/apply');
    const profit=contracts.endpoints.find((endpoint)=>routeKey(endpoint)==='GET /v1/profit/report');
    expect(run?.requestBody).toEqual(['marketplaceListingId','priceRuleId','trigger']);
    expect(run?.idempotencyKey).toBe(true);
    expect(apply?.idempotencyKey).toBe(true);
    expect(profit?.query).toEqual(['from','to']);
  });
});

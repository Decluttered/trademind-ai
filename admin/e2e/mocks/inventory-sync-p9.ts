import { ok } from './envelope';

export const p9Run = {
  id: 'run-p9-e2e-1',
  shopConnectionId: 'shop-p9-e2e',
  platform: 'douyin',
  providerMode: 'mock',
  status: 'succeeded',
  triggerType: 'manual',
  fixtureScenario: 'success_single_page',
  statistics: {
    totalRecordCount: 2,
    matchedRecordCount: 1,
    unmatchedRecordCount: 1,
    conflictRecordCount: 0,
    failedRecordCount: 0,
    manualBindingRequestCount: 1,
    confirmedBindingCount: 1,
    pagesProcessed: 1,
  },
  cursorHash: 'hash-p9-e2e',
  revision: 2,
  startedAt: '2026-08-04T09:00:00.000Z',
  finishedAt: '2026-08-04T09:00:01.000Z',
  createdAt: '2026-08-04T09:00:00.000Z',
  updatedAt: '2026-08-04T09:00:01.000Z',
  allowedActions: { canViewSnapshots: true, canRerun: true, canViewAudit: true },
};

export const p9Snapshot = {
  id: 'snapshot-p9-e2e-1',
  inventorySyncRunId: p9Run.id,
  shopConnectionId: p9Run.shopConnectionId,
  platform: 'douyin',
  externalProductId: 'dy-product-1',
  externalSkuId: 'dy-sku-1',
  externalProductCode: 'DP-1',
  externalSkuCode: 'DS-1',
  productTitle: 'Fixture Product',
  variantTitle: 'Fixture Variant',
  availableQuantity: 10,
  reservedQuantity: 1,
  totalQuantity: 11,
  observedAt: '2026-08-04T09:00:00.000Z',
  binding: {
    result: 'manual_review',
    bindingId: 'binding-p9-e2e-1',
    bindingStatus: 'proposed',
    localProductId: 'product-local-1',
    localSkuId: 'sku-local-1',
    confidence: 8500,
    calibrationVersion: 1,
    manualRequestId: 'manual-p9-e2e-1',
  },
  createdAt: '2026-08-04T09:00:00.000Z',
};

export const p9Binding = {
  id: 'binding-p9-e2e-1',
  shopConnectionId: p9Run.shopConnectionId,
  platform: 'douyin',
  externalProductId: 'dy-product-1',
  externalSkuId: 'dy-sku-1',
  externalSkuCode: 'DS-1',
  localProductId: 'product-local-1',
  localSkuId: 'sku-local-1',
  bindingSource: 'manual',
  bindingStatus: 'confirmed',
  confidence: 8500,
  calibrationVersion: 1,
  calibrationReason: 'manual_review_required',
  confirmedBy: 'admin-e2e',
  confirmedAt: '2026-08-04T09:01:00.000Z',
  revision: 3,
  createdAt: '2026-08-04T09:00:00.000Z',
  updatedAt: '2026-08-04T09:01:00.000Z',
  allowedActions: { canViewHistory: true, canViewCalibration: true },
};

export const p9Calibration = {
  id: 'calibration-p9-e2e-1',
  inventorySyncRunId: p9Run.id,
  inventorySnapshotItemId: p9Snapshot.id,
  externalSkuId: 'dy-sku-1',
  candidateLocalProductId: 'product-local-1',
  candidateLocalSkuId: 'sku-local-1',
  matchStrategy: 'normalized_sku_code',
  confidence: 8500,
  scoreBreakdown: [{ signal: 'normalizedSKUCodeScore', score: 8500 }],
  reasonCodes: ['manual_review_required'],
  calibrationVersion: 1,
  status: 'candidate',
  createdAt: '2026-08-04T09:00:00.000Z',
};

export const p9ManualRequest = {
  id: 'manual-p9-e2e-1',
  inventorySyncRunId: p9Run.id,
  inventorySnapshotItemId: p9Snapshot.id,
  shopConnectionId: p9Run.shopConnectionId,
  externalSkuId: 'dy-sku-1',
  status: 'pending',
  reasonCode: 'manual_review_required',
  candidateCount: 1,
  suggestedLocalSkuId: 'sku-local-1',
  revision: 4,
  createdAt: '2026-08-04T09:00:00.000Z',
  updatedAt: '2026-08-04T09:00:00.000Z',
  allowedActions: { canConfirm: true, canReject: true },
};

const manualDetail = {
  request: p9ManualRequest,
  decisions: [],
};

export function inventorySyncP9Response(path: string) {
  if (path === '/api/v1/product-skus/search') {
    return ok({ list: [{ productId: 'product-local-1', productTitle: 'Local Product', productSkuId: 'sku-local-1', skuCode: 'SKU-1', skuName: 'Local SKU' }] });
  }
  if (path === '/api/v1/inventory-sync/runs') return ok({ items: [p9Run], nextCursor: '', hasMore: false, limit: 20 });
  if (path === `/api/v1/inventory-sync/runs/${p9Run.id}`) return ok(p9Run);
  if (path === `/api/v1/inventory-sync/runs/${p9Run.id}/snapshots`) return ok({ items: [p9Snapshot], nextCursor: '', hasMore: false, limit: 20 });
  if (path === `/api/v1/inventory-sync/runs/${p9Run.id}/audit-events`) {
    return ok({
      items: [{
        id: 'audit-p9-e2e-1',
        action: 'inventory_sync.run_created',
        resource: 'inventory_sync_run',
        resourceId: p9Run.id,
        shopId: p9Run.shopConnectionId,
        platform: 'douyin',
        permission: 'inventory_sync.read',
        requestId: 'request-p9-e2e',
        status: 'success',
        metadata: { providerMode: 'mock', fixtureScenario: 'success_single_page', token: 'should-not-render' },
        actorId: 'admin-e2e',
        actorRole: 'admin',
        createdAt: '2026-08-04T09:00:00.000Z',
      }],
      nextCursor: '',
      hasMore: false,
      limit: 20,
    });
  }
  if (path === `/api/v1/inventory-sync/snapshots/${p9Snapshot.id}`) return ok(p9Snapshot);
  if (path === `/api/v1/inventory-sync/snapshots/${p9Snapshot.id}/calibrations`) return ok({ items: [p9Calibration], nextCursor: '', hasMore: false, limit: 20 });
  if (path === '/api/v1/inventory-sync/bindings') return ok({ items: [p9Binding], nextCursor: '', hasMore: false, limit: 20 });
  if (path === `/api/v1/inventory-sync/bindings/${p9Binding.id}`) return ok(p9Binding);
  if (path === `/api/v1/inventory-sync/bindings/${p9Binding.id}/history`) {
    return ok({ binding: p9Binding, calibrations: [p9Calibration], manualDecisions: [] });
  }
  if (path === '/api/v1/inventory-sync/manual-binding-requests') return ok({ items: [p9ManualRequest], nextCursor: '', hasMore: false, limit: 20 });
  if (path === `/api/v1/inventory-sync/manual-binding-requests/${p9ManualRequest.id}`) return ok(manualDetail);
  return null;
}

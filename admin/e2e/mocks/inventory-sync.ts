import { ok } from './envelope';

export const inventoryRun = {
  id: 'inventory-run-e2e-1',
  shopConnectionId: 'inventory-shop-e2e',
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
  cursorHash: 'inventory-hash-e2e',
  revision: 2,
  startedAt: '2026-08-04T09:00:00.000Z',
  finishedAt: '2026-08-04T09:00:01.000Z',
  createdAt: '2026-08-04T09:00:00.000Z',
  updatedAt: '2026-08-04T09:00:01.000Z',
  allowedActions: { canViewSnapshots: true, canRerun: true, canViewAudit: true },
};

export const inventorySnapshot = {
  id: 'inventory-snapshot-e2e-1',
  inventorySyncRunId: inventoryRun.id,
  shopConnectionId: inventoryRun.shopConnectionId,
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
    bindingId: 'inventory-binding-e2e-1',
    bindingStatus: 'proposed',
    localProductId: 'product-local-1',
    localSkuId: 'sku-local-1',
    confidence: 8500,
    calibrationVersion: 1,
    manualRequestId: 'inventory-manual-e2e-1',
  },
  createdAt: '2026-08-04T09:00:00.000Z',
};

export const inventoryBinding = {
  id: 'inventory-binding-e2e-1',
  shopConnectionId: inventoryRun.shopConnectionId,
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

export const inventoryCalibration = {
  id: 'inventory-calibration-e2e-1',
  inventorySyncRunId: inventoryRun.id,
  inventorySnapshotItemId: inventorySnapshot.id,
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

export const manualBindingRequest = {
  id: 'inventory-manual-e2e-1',
  inventorySyncRunId: inventoryRun.id,
  inventorySnapshotItemId: inventorySnapshot.id,
  shopConnectionId: inventoryRun.shopConnectionId,
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
  request: manualBindingRequest,
  decisions: [],
};

export function inventorySyncResponse(path: string) {
  if (path === '/api/v1/product-skus/search') {
    return ok({ list: [{ productId: 'product-local-1', productTitle: 'Local Product', productSkuId: 'sku-local-1', skuCode: 'SKU-1', skuName: 'Local SKU' }] });
  }
  if (path === '/api/v1/inventory-sync/runs') return ok({ items: [inventoryRun], nextCursor: '', hasMore: false, limit: 20 });
  if (path === `/api/v1/inventory-sync/runs/${inventoryRun.id}`) return ok(inventoryRun);
  if (path === `/api/v1/inventory-sync/runs/${inventoryRun.id}/snapshots`) return ok({ items: [inventorySnapshot], nextCursor: '', hasMore: false, limit: 20 });
  if (path === `/api/v1/inventory-sync/runs/${inventoryRun.id}/audit-events`) {
    return ok({
      items: [{
        id: 'inventory-audit-e2e-1',
        action: 'inventory_sync.run_created',
        resource: 'inventory_sync_run',
        resourceId: inventoryRun.id,
        shopId: inventoryRun.shopConnectionId,
        platform: 'douyin',
        permission: 'inventory_sync.read',
        requestId: 'inventory-request-e2e',
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
  if (path === `/api/v1/inventory-sync/snapshots/${inventorySnapshot.id}`) return ok(inventorySnapshot);
  if (path === `/api/v1/inventory-sync/snapshots/${inventorySnapshot.id}/calibrations`) return ok({ items: [inventoryCalibration], nextCursor: '', hasMore: false, limit: 20 });
  if (path === '/api/v1/inventory-sync/bindings') return ok({ items: [inventoryBinding], nextCursor: '', hasMore: false, limit: 20 });
  if (path === `/api/v1/inventory-sync/bindings/${inventoryBinding.id}`) return ok(inventoryBinding);
  if (path === `/api/v1/inventory-sync/bindings/${inventoryBinding.id}/history`) {
    return ok({ binding: inventoryBinding, calibrations: [inventoryCalibration], manualDecisions: [] });
  }
  if (path === '/api/v1/inventory-sync/manual-binding-requests') return ok({ items: [manualBindingRequest], nextCursor: '', hasMore: false, limit: 20 });
  if (path === `/api/v1/inventory-sync/manual-binding-requests/${manualBindingRequest.id}`) return ok(manualDetail);
  return null;
}

import { request } from '@umijs/max';
import { describe, expect, it, vi } from 'vitest';
import {
  confirmManualBindingRequest,
  createInventorySyncRun,
  getInventoryBindingHistory,
  listInventoryBindings,
  listInventoryRunAuditEvents,
  listInventorySnapshots,
  listInventorySyncRuns,
  listManualBindingRequests,
  recalibrateInventorySnapshot,
  rejectManualBindingRequest,
  rerunInventorySyncRun,
} from '../inventorySync';

const requestMock = vi.mocked(request);

describe('Inventory sync API service', () => {
  it('uses cursor list contracts without page totals', async () => {
    requestMock.mockResolvedValue({ code: 0, message: 'ok', data: { items: [], hasMore: false, limit: 20 } });

    await listInventorySyncRuns({ shopConnectionId: 'shop-1', status: 'succeeded', providerMode: 'mock', cursor: 'next', limit: 20 });
    await listInventorySnapshots('run/1', { bindingResult: 'manual_review', cursor: 'snap', limit: 20 });
    await listInventoryBindings({ shopConnectionId: 'shop-1', bindingStatus: 'confirmed', bindingSource: 'manual', cursor: 'bind', limit: 20 });
    await listManualBindingRequests({ shopConnectionId: 'shop-1', status: 'pending', cursor: 'manual', limit: 20 });
    await listInventoryRunAuditEvents('run/1', { cursor: 'audit', limit: 20 });

    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/runs', {
      method: 'GET',
      params: { shopConnectionId: 'shop-1', status: 'succeeded', providerMode: 'mock', cursor: 'next', limit: 20 },
    });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/runs/run%2F1/snapshots', {
      method: 'GET',
      params: { bindingResult: 'manual_review', cursor: 'snap', limit: 20 },
    });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/bindings', {
      method: 'GET',
      params: { shopConnectionId: 'shop-1', bindingStatus: 'confirmed', bindingSource: 'manual', cursor: 'bind', limit: 20 },
    });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/manual-binding-requests', {
      method: 'GET',
      params: { shopConnectionId: 'shop-1', status: 'pending', cursor: 'manual', limit: 20 },
    });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/runs/run%2F1/audit-events', {
      method: 'GET',
      params: { cursor: 'audit', limit: 20 },
    });
  });

  it('sends exact write bodies with idempotency keys', async () => {
    requestMock.mockResolvedValue({ code: 0, message: 'ok', data: { id: 'ok' } });

    await createInventorySyncRun(
      { shopConnectionId: 'shop-1', platform: 'douyin', providerMode: 'mock', fixtureScenario: 'success_single_page' },
      'key-create',
    );
    await rerunInventorySyncRun('run-1', { expectedRevision: 3 }, 'key-rerun');
    await recalibrateInventorySnapshot('snapshot-1', { expectedCalibrationVersion: 2, reason: 'manual check' }, 'key-recalibrate');
    await confirmManualBindingRequest('request-1', { expectedRevision: 4, selectedLocalSkuId: 'sku-1', comment: 'ok' }, 'key-confirm');
    await rejectManualBindingRequest('request-2', { expectedRevision: 5, reasonCode: 'no_binding_candidate', comment: 'reject' }, 'key-reject');

    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/runs', {
      method: 'POST',
      data: { shopConnectionId: 'shop-1', platform: 'douyin', providerMode: 'mock', fixtureScenario: 'success_single_page' },
      headers: { 'Idempotency-Key': 'key-create' },
    });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/runs/run-1/rerun', {
      method: 'POST',
      data: { expectedRevision: 3 },
      headers: { 'Idempotency-Key': 'key-rerun' },
    });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/snapshots/snapshot-1/recalibrate', {
      method: 'POST',
      data: { expectedCalibrationVersion: 2, reason: 'manual check' },
      headers: { 'Idempotency-Key': 'key-recalibrate' },
    });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/manual-binding-requests/request-1/confirm', {
      method: 'POST',
      data: { expectedRevision: 4, selectedLocalSkuId: 'sku-1', comment: 'ok' },
      headers: { 'Idempotency-Key': 'key-confirm' },
    });
    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/manual-binding-requests/request-2/reject', {
      method: 'POST',
      data: { expectedRevision: 5, reasonCode: 'no_binding_candidate', comment: 'reject' },
      headers: { 'Idempotency-Key': 'key-reject' },
    });
  });

  it('uses binding history endpoint directly', async () => {
    requestMock.mockResolvedValueOnce({ code: 0, message: 'ok', data: { binding: { id: 'binding-1' }, calibrations: [], manualDecisions: [] } });

    await getInventoryBindingHistory('binding/1');

    expect(requestMock).toHaveBeenCalledWith('/api/v1/inventory-sync/bindings/binding%2F1/history', { method: 'GET' });
  });
});

import { request } from '@umijs/max';
import { describe, expect, it, vi } from 'vitest';
import { getProductionRuntimeStatus, productionDraftBlockReason } from '../productionControl';

const requestMock = vi.mocked(request);

describe('production control service', () => {
  it('reads the authenticated P10 runtime status', async () => {
    requestMock.mockResolvedValueOnce({ code: 0, message: 'ok', data: { productionReady: false } });

    await getProductionRuntimeStatus();

    expect(requestMock).toHaveBeenCalledWith('/api/v1/p10/status', { method: 'GET' });
  });

  it('fails closed when runtime status is unavailable or write kill is active', () => {
    expect(productionDraftBlockReason()).toContain('禁止真实平台写入');
    expect(productionDraftBlockReason({
      currentAllowedLevel: 'L3',
      environment: 'production',
      realProviderEnabled: true,
      realPlatformNetworkEnabled: true,
      realCredentialsEnabled: true,
      realProductDraftWriteEnabled: true,
      backgroundWorkerEnabled: true,
      productPublishQueueEnabled: true,
      providerWriteReady: false,
      automaticRetryEnabled: false,
      control: { providerKillActive: false, tenantKillActive: false, shopKillActive: false, readKillActive: true, writeKillActive: true, revision: 1 },
      initialLimits: {},
      productionReady: false,
      productionAcceptancePassed: true,
    })).toContain('写入开关');
  });

  it('blocks production drafts when the provider runtime has not allowed writes', () => {
    expect(productionDraftBlockReason({
      currentAllowedLevel: 'L3',
      environment: 'production',
      realProviderEnabled: true,
      realPlatformNetworkEnabled: true,
      realCredentialsEnabled: true,
      realProductDraftWriteEnabled: true,
      backgroundWorkerEnabled: true,
      productPublishQueueEnabled: true,
      providerWriteReady: false,
      automaticRetryEnabled: false,
      control: { providerKillActive: false, tenantKillActive: false, shopKillActive: false, readKillActive: true, writeKillActive: false, revision: 2 },
      allowlist: { tenantId: 101, shopId: 'shop-1', enabled: true, revision: 1 },
      gray: { tenantId: 101, shopId: 'shop-1', maxSku: 100, status: 'active', ownerApproved: true, technicalLeadApproved: true, revision: 4 },
      initialLimits: { maxTenant: 1, maxShop: 1, maxSku: 100 },
      productionReady: false,
      productionAcceptancePassed: true,
    })).toContain('平台运行开关、草稿功能或平台灰度店铺尚未放行');
  });

  it('allows no block reason only for backend-confirmed production readiness', () => {
    expect(productionDraftBlockReason({
      currentAllowedLevel: 'L3',
      environment: 'production',
      realProviderEnabled: true,
      realPlatformNetworkEnabled: true,
      realCredentialsEnabled: true,
      realProductDraftWriteEnabled: true,
      backgroundWorkerEnabled: true,
      productPublishQueueEnabled: true,
      providerWriteReady: true,
      automaticRetryEnabled: false,
      control: { providerKillActive: false, tenantKillActive: false, shopKillActive: false, readKillActive: true, writeKillActive: false, revision: 2 },
      allowlist: { tenantId: 101, shopId: 'shop-1', enabled: true, revision: 1 },
      gray: { tenantId: 101, shopId: 'shop-1', maxSku: 100, status: 'active', ownerApproved: true, technicalLeadApproved: true, revision: 4 },
      initialLimits: { maxTenant: 1, maxShop: 1, maxSku: 100 },
      productionReady: true,
      productionAcceptancePassed: true,
    })).toBeUndefined();
  });
});

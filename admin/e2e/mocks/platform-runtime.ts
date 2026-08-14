import { ok } from "./envelope";

const runtimeStatus = {
  status: "normal",
  message: "后台任务调用处于正常运行状态",
  changedAt: "2026-08-14T04:03:53Z",
};

const health = {
  overallStatus: "degraded",
  overallLabel: "部分检查项需要处理",
  checkedAt: "2026-08-14T04:03:53Z",
  config: {
    status: "unhealthy",
    label: "平台应用配置不完整",
    details: { missingFields: 2 },
  },
  auth: { status: "healthy", label: "店铺授权正常" },
  storage: { status: "degraded", label: "存储公网访问需要复核" },
  tasks: {
    status: "degraded",
    label: "存在待处理任务",
    details: {
      failedPending: 2,
      recoveryRequired: 1,
      resultUnknown: 0,
      stale24h: 0,
    },
  },
  api: { status: "healthy", label: "接口调用稳定" },
  runtime: runtimeStatus,
  grayRelease: {
    enabled: false,
    writeOperationsEnabled: false,
    scheduledOrderSyncEnabled: false,
    scheduledInventorySyncEnabled: false,
    shopIds: [],
  },
};

const metrics = {
  generatedAt: "2026-08-14T04:03:53Z",
  apiRequestsTotal: 1280,
  apiSuccessTotal: 1264,
  apiFailedTotal: 16,
  apiSuccessRate: 98.75,
  apiDurationAvgMs: 182,
  apiTimeoutTotal: 1,
  apiRateLimitedTotal: 2,
  apiRetryTotal: 3,
  tokenRefreshTotal: 8,
  tokenRefreshFailedTotal: 0,
  runtimeBlockedTasksTotal: 0,
  staleTasksTotal: 0,
  recoverySuccessTotal: 4,
  recoveryFailedTotal: 0,
  productDraftCreateTotal: 38,
  productDraftCreateFailedTotal: 1,
  imageUploadTotal: 92,
  imageUploadFailedTotal: 0,
  skuAutoBoundTotal: 44,
  skuManualBoundTotal: 3,
  skuUnmatchedTotal: 0,
  skuAmbiguousTotal: 0,
  orderFetchedTotal: 156,
  orderCreatedTotal: 120,
  orderUpdatedTotal: 36,
  orderPartialSuccessTotal: 0,
  orderUnmatchedItemsTotal: 0,
  orderInventoryDeductedTotal: 118,
  inventorySyncTotal: 64,
  inventorySyncSuccessTotal: 63,
  inventorySyncFailedTotal: 1,
  inventorySyncSkippedTotal: 0,
  failureTasksPending: 2,
  authorizationsExpiring: 0,
};

const releaseGate = {
  overallConclusion: "Release Candidate",
  checkedAt: "2026-08-14T04:03:53Z",
  items: [
    {
      key: "config",
      label: "应用配置",
      status: "failed",
      message: "补全应用配置后重新运行健康检查。",
    },
    {
      key: "credentials",
      label: "真实凭证联调",
      status: "blocked",
      message: "需要在受控人工验收中完成。",
    },
    {
      key: "storage",
      label: "存储公网访问",
      status: "warning",
      message: "请复核商品图片公网访问状态。",
    },
  ],
};

export function platformRuntimeResponse(path: string) {
  if (path === "/api/v1/platform/douyin/health") return ok(health);
  if (path === "/api/v1/platform/douyin/metrics-summary") return ok(metrics);
  if (path === "/api/v1/platform/douyin/release-gate") return ok(releaseGate);
  if (path === "/api/v1/platform/douyin/runtime-status")
    return ok(runtimeStatus);
  return null;
}

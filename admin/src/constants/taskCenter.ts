/** Failed task / alert severity levels (matches backend failureclassifier) */
export const TASK_FAILURE_SEVERITY: Record<string, { text: string; color: string }> = {
  low: { text: 'Low', color: 'default' },
  medium: { text: 'Medium', color: 'blue' },
  high: { text: 'High', color: 'orange' },
  critical: { text: 'Critical', color: 'red' },
};

export const TASK_FAILURE_SEVERITY_OPTIONS = Object.entries(TASK_FAILURE_SEVERITY).map(([value, m]) => ({
  label: m.text,
  value,
}));

/** Failure category (matches backend failureclassifier category constants) */
export const TASK_FAILURE_CATEGORY_LABEL: Record<string, string> = {
  platform_auth: 'Platform authorization failed',
  platform_permission: 'Insufficient platform permission',
  platform_rate_limit: 'Platform rate limit reached',
  platform_api_error: 'Platform API error',
  platform_config_incomplete: 'Platform configuration is incomplete',
  network_timeout: 'Network timeout',
  collector_blocked: 'Collection was blocked',
  collector_platform_login: 'Platform login or verification expired',
  login_required: 'Login required',
  collector_missing_images: 'Images are missing',
  collector_missing_price: 'Price field is missing',
  collector_evaluate_script: 'Collection script failed',
  collector_invalid_url: 'Collection link is invalid',
  ai_provider_error: 'AI provider error',
  ai_config_incomplete: 'AI configuration is incomplete',
  image_provider_error: 'Image provider error',
  storage_error: 'Storage error',
  validation_error: 'Validation failed',
  inventory_mapping_missing: 'Variant mapping is missing',
  inventory_deduct_failed: 'Inventory deduction failed',
  inventory_sync_failed: 'Inventory sync failed',
  inventory_sync_partial_success: 'Inventory sync partially succeeded',
  inventory_sku_not_bound: 'SKU is not bound',
  inventory_sku_ambiguous: 'SKU binding is ambiguous',
  inventory_stock_invalid: 'Inventory value is invalid',
  inventory_platform_permission_denied: 'Insufficient platform inventory permission',
  customer_reply_generate_failed: 'AI reply suggestion generation failed',
  customer_reply_send_failed: 'Customer reply failed to send',
  customer_reply_permission_denied: 'Insufficient customer-message permission',
  customer_platform_not_authorized: 'Platform is not authorized',
  customer_message_sync_failed: 'Customer-message sync failed',
  customer_message_sync_partial_success: 'Customer-message sync partially succeeded',
  customer_conversation_not_found: 'Conversation was not found',
  inventory_product_not_bound: 'Platform product is not bound',
  inventory_platform_sku_missing: 'Platform SKU is missing',
  sku_mapping_missing: 'Variant mapping is missing',
  worker_lease_expired: 'Background task timed out',
  system_error: 'System error',
  unknown: 'Unknown',
  // AI batch product copy (aiproducttext)
  ai_text_generation_failed: 'AI copy generation failed',
  ai_text_apply_conflict: 'A content conflict was found while applying AI copy',
  ai_text_apply_failed: 'Applying AI copy failed',
  ai_text_undo_failed: 'Undoing AI copy failed',
  ai_text_quality_warning: 'AI copy suggestion needs review',
  ai_image_process_failed: 'AI image processing failed',
  ai_image_apply_conflict: 'A conflict was found while applying AI images',
  ai_image_apply_failed: 'Applying AI images failed',
  ai_image_undo_failed: 'Undoing AI images failed',
  ai_image_quality_warning: 'AI image quality warning',
  ai_image_provider_config_missing: 'AI image provider is not configured',
  ai_image_dashscope_key_missing: 'DashScope key is not configured',
  ai_image_storage_public_url_missing: 'Public storage URL is not configured',
  ai_image_download_failed: 'Source AI image could not be downloaded',
  ai_image_unsupported_operation: 'AI image operation is not supported',
  // Douyin Shop platform-level in-app alerts (douyinruntime/alert.go)
  douyin_token_refresh_failed: 'Access token refresh failed',
  douyin_webhook_shop_not_resolved: 'Webhook shop was not resolved',
  douyin_webhook_shop_ambiguous: 'Webhook shop binding is ambiguous',
  douyin_webhook_binding_mismatch: 'Webhook binding does not match',
  douyin_webhook_tenant_mismatch: 'Webhook tenant does not match',
  douyin_webhook_authorization_expired: 'Webhook shop authorization expired',
  douyin_webhook_binding_revoked: 'Webhook shop binding was revoked',
  douyin_webhook_app_binding_mismatch: 'Webhook app binding does not match',
  douyin_auth_expiring: 'Shop authorization is expiring soon',
  douyin_auth_expired: 'Shop authorization expired',
  douyin_auth_need_check: 'Shop authorization needs review',
  douyin_product_draft_failures: 'Product draft failure backlog',
  douyin_product_result_unknown: 'Product result cannot yet be confirmed',
  douyin_product_recovery_failed: 'Product task recovery failed',
  douyin_image_upload_failure_rate: 'Image upload failure rate is too high',
  douyin_storage_public_failed: 'Public storage access failed',
  douyin_order_sync_failed: 'Order sync failed',
  douyin_order_partial_stale: 'Order sync is partially stale',
  douyin_inventory_sync_failed: 'Inventory sync failed',
  douyin_inventory_stale: 'Inventory sync is stale',
  douyin_runtime_emergency_disabled: 'Douyin Shop emergency shutdown',
  douyin_stale_tasks_high: 'Too many stale tasks',
  douyin_failure_backlog: 'Failed-task backlog',
  douyin_rate_limit_spike: 'Platform rate-limit spike',
};

export function failureCategoryLabel(cat?: string): string {
  const k = (cat || '').trim();
  if (!k) return '—';
  return TASK_FAILURE_CATEGORY_LABEL[k] || k;
}

export function failureSeverityLabel(sev?: string): string {
  const k = (sev || '').trim().toLowerCase();
  if (!k) return '—';
  return TASK_FAILURE_SEVERITY[k]?.text || k;
}

/** Task types whose detail can be queried in the failed task center (matches backend parseTaskType) */
export const TASK_CENTER_FAILURE_TASK_TYPES = [
  'collect',
  'image',
  'order_sync',
  'customer_message_sync',
  'customer_failure',
  'product_publish',
  'inventory_sync',
  'ai_text',
  'ai_image',
] as const;

export type TaskCenterFailureTaskType = (typeof TASK_CENTER_FAILURE_TASK_TYPES)[number];

/** Platform-level in-app alert taskType (sourceId is not a business task UUID, cannot use the failure detail endpoint) */
export const PLATFORM_ALERT_TASK_TYPES = ['douyin_platform'] as const;

const TASK_FAILURE_DETAIL_ID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** Task type (failed task center list) */
export const TASK_CENTER_TASK_TYPE_LABEL: Record<string, string> = {
  collect: 'Collection',
  image: 'AI image processing',
  order_sync: 'Order sync',
  customer_message_sync: 'Customer-message sync',
  customer_failure: 'Customer-service issue',
  product_publish: 'Product publishing',
  inventory_sync: 'Inventory sync',
  ai_text: 'AI batch copy',
  ai_image: 'AI batch images',
  douyin_platform: 'Douyin Shop platform alert',
};

export function isTaskCenterFailureTaskType(taskType?: string | null): boolean {
  const k = (taskType || '').trim().toLowerCase();
  return (TASK_CENTER_FAILURE_TASK_TYPES as readonly string[]).includes(k);
}

export function isPlatformAlertTaskType(taskType?: string | null): boolean {
  const k = (taskType || '').trim().toLowerCase();
  return (PLATFORM_ALERT_TASK_TYPES as readonly string[]).includes(k);
}

export function isTaskFailureDetailId(id?: string | null): boolean {
  const k = (id || '').trim();
  return TASK_FAILURE_DETAIL_ID_RE.test(k);
}

/** Whether GET /task-center/failures/:taskType/:id can be opened */
export function canOpenFailureDetail(taskType?: string | null, sourceId?: string | null): boolean {
  return isTaskCenterFailureTaskType(taskType) && isTaskFailureDetailId(sourceId);
}

/** Alert center "related entry" deep link: platform alerts go to the ops page, business failed tasks go to the detail deep link */
export function resolveAlertRelatedLink(alert: {
  taskType: string;
  sourceId: string;
  platform?: string;
}): { href: string; label: string } {
  const taskType = (alert.taskType || '').trim();
  const sourceId = (alert.sourceId || '').trim();

  if (taskType === 'douyin_platform') {
    return { href: '/ops/platform-runtime?platform=douyin_shop', label: 'Platform operations' };
  }
  if (isPlatformAlertTaskType(taskType)) {
    return { href: '/ops/task-center/alerts', label: 'Alert center' };
  }
  if (canOpenFailureDetail(taskType, sourceId)) {
    const sp = new URLSearchParams({ taskType, jumpId: sourceId });
    return { href: `/ops/task-center/failures?${sp.toString()}`, label: 'Failed tasks' };
  }
  if ((alert.platform || '').trim()) {
    const sp = new URLSearchParams({ platform: alert.platform!.trim() });
    return { href: `/ops/task-center/failures?${sp.toString()}`, label: 'Failed tasks' };
  }
  return { href: '/ops/task-center/failures', label: 'Failed tasks' };
}

/** Douyin Shop task recovery status → user-visible copy (does not display internal values like stale / result_unknown) */
export const TASK_RECOVERY_STATUS_LABEL: Record<string, string> = {
  stale: 'Task has been running too long',
  result_unknown: 'Platform result cannot yet be confirmed',
  recovery_required: 'Manual review required',
  recovery_failed: 'Recovery failed',
  superseded: 'Replaced by a newer task',
  skipped: 'Skipped',
};

export const TASK_RECOVERY_STATUS_OPTIONS = Object.entries(TASK_RECOVERY_STATUS_LABEL).map(
  ([value, label]) => ({ value, label }),
);

export function recoveryStatusLabel(status?: string | null): string {
  const k = (status ?? '').trim();
  if (!k) return '—';
  return TASK_RECOVERY_STATUS_LABEL[k] || '—';
}

/** Worker process effective status (monitoring page) */
export const WORKER_EFFECTIVE_STATUS: Record<string, { text: string; color: string }> = {
  running: { text: 'Running', color: 'success' },
  stale: { text: 'Heartbeat timed out', color: 'warning' },
  stopped: { text: 'Stopped', color: 'default' },
};

export const WORKER_STATUS_METRIC: Record<
  'running' | 'stale' | 'stopped',
  { text: string; valueStyle: string }
> = {
  running: { text: 'Running', valueStyle: 'var(--ant-color-success)' },
  stale: { text: 'Heartbeat timed out', valueStyle: 'var(--ant-color-warning)' },
  stopped: { text: 'Stopped', valueStyle: 'var(--ant-color-text-secondary)' },
};

/** Worker monitoring grouped by type (matches backend byType keys) */
export const WORKER_MONITOR_TYPE_KEYS = [
  'collect',
  'image',
  'order_sync',
  'customer_message_sync',
  'customer_failure',
  'product_publish',
  'inventory_sync',
  'ai_text',
] as const;

export type WorkerMonitorTypeKey = (typeof WORKER_MONITOR_TYPE_KEYS)[number];

export function workerTypeLabel(type?: string): string {
  const k = (type || '').trim();
  if (!k) return '—';
  return TASK_CENTER_TASK_TYPE_LABEL[k] || k;
}

/** Alert center task type display name (matches workerTypeLabel) */
export function taskCenterTaskTypeLabel(taskType?: string): string {
  return workerTypeLabel(taskType);
}

/** Normalized status */
export const TASK_NORMALIZED_STATUS: Record<string, { text: string; color: string }> = {
  failed: { text: 'Failed', color: 'error' },
  partial_success: { text: 'Partially succeeded', color: 'warning' },
  retrying: { text: 'Retrying', color: 'processing' },
  stale: { text: 'Stale', color: 'warning' },
  lease_expired: { text: 'Timed out', color: 'warning' },
  running: { text: 'Running', color: 'processing' },
  pending: { text: 'Queued', color: 'default' },
  success: { text: 'Succeeded', color: 'success' },
  cancelled: { text: 'Cancelled', color: 'default' },
};

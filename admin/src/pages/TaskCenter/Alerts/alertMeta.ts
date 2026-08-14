export type AlertSource = 'business' | 'system';

type SourceAccess = {
  business: boolean;
  system: boolean;
};

export function resolveAlertSource(raw: string | null | undefined, access: SourceAccess): AlertSource {
  if (raw === 'business' && access.business) return 'business';
  if (raw === 'system' && access.system) return 'system';
  return access.business ? 'business' : 'system';
}

export const SYSTEM_ALERT_SEVERITY_META: Record<string, { color: string; text: string }> = {
  info: { color: 'blue', text: '提示' },
  warning: { color: 'orange', text: '警告' },
  critical: { color: 'red', text: '严重' },
};

export const SYSTEM_ALERT_STATUS_META: Record<string, { color: string; text: string }> = {
  firing: { color: 'red', text: '告警中' },
  acknowledged: { color: 'blue', text: '已确认' },
  silenced: { color: 'default', text: '已静默' },
  resolved: { color: 'green', text: '已恢复' },
  expired: { color: 'default', text: '已过期' },
};

export function systemAlertSeverityMeta(value?: string) {
  const key = (value || '').trim().toLowerCase();
  return SYSTEM_ALERT_SEVERITY_META[key] ?? { color: 'default', text: key || '-' };
}

export function systemAlertStatusMeta(value?: string) {
  const key = (value || '').trim().toLowerCase();
  return SYSTEM_ALERT_STATUS_META[key] ?? { color: 'default', text: key || '-' };
}

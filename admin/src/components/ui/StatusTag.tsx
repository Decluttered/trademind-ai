import { Tag } from 'antd';
import { commonStatusLabel } from '@/constants/copywriting';
import { COLLECT_TASK_STATUS } from '@/constants/status';

type StatusColor = 'default' | 'processing' | 'success' | 'error' | 'warning' | 'blue' | 'cyan';

export type StatusTagProps = {
  status?: string | null;
  /** Directly specify the text, takes priority over the status mapping */
  text?: string;
  color?: StatusColor;
  className?: string;
};

const STATUS_COLOR_MAP: Record<string, StatusColor> = {
  pending: 'processing',
  running: 'processing',
  success: 'success',
  partial_success: 'warning',
  failed: 'error',
  cancelled: 'default',
  enabled: 'success',
  disabled: 'default',
  configured: 'success',
  unconfigured: 'default',
  authorized: 'success',
  expired: 'warning',
  need_check: 'warning',
  bound: 'success',
  unmatched: 'default',
  ambiguous: 'warning',
  skipped: 'default',
};

/** Unified status Tag */
export default function StatusTag({ status, text, color, className }: StatusTagProps) {
  const k = (status ?? '').trim().toLowerCase();
  const collectMeta = k ? COLLECT_TASK_STATUS[k as keyof typeof COLLECT_TASK_STATUS] : undefined;
  const label = text ?? collectMeta?.text ?? commonStatusLabel(status);
  const tagColor = color ?? collectMeta?.color ?? STATUS_COLOR_MAP[k] ?? 'default';
  return (
    <Tag color={tagColor as never} className={className}>
      {label}
    </Tag>
  );
}

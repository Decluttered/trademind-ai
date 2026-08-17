import { Typography } from 'antd';

/** JSON display inside the task detail drawer */
export function formatTaskJson(value: unknown): string {
  if (value == null || value === '') return '—';
  try {
    return typeof value === 'string' ? value : JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export type TaskJsonBlockProps = {
  title: string;
  value: unknown;
  maxHeight?: number;
  /** Whether this is the last item within the collapsible section (removes paragraph bottom margin) */
  last?: boolean;
};

export default function TaskJsonBlock({ title, value, maxHeight = 200, last }: TaskJsonBlockProps) {
  return (
    <Typography.Paragraph style={{ marginBottom: last ? 0 : 8 }}>
      <Typography.Text strong>{title}</Typography.Text>
      <pre
        style={{
          fontSize: 12,
          overflow: 'auto',
          maxHeight,
          whiteSpace: 'pre-wrap',
          marginTop: 6,
          marginBottom: 0,
        }}
      >
        {formatTaskJson(value)}
      </pre>
    </Typography.Paragraph>
  );
}

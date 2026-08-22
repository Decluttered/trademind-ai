import { ProCard } from '@ant-design/pro-components';
import { TmPageContainer, TmProTable as ProTable } from '@/components/ui';
import { commonStatusLabel } from '@/constants/copywriting';
import { formatDateTime } from '@/utils/formatTime';
import type { ProColumns } from '@ant-design/pro-components';
import { history } from '@umijs/max';
import { Button, Col, Row, Space, Statistic, Tag, Typography } from 'antd';
import { useEffect, useMemo, useState } from 'react';
import {
  TASK_CENTER_TASK_TYPE_LABEL,
  WORKER_EFFECTIVE_STATUS,
  WORKER_MONITOR_TYPE_KEYS,
  WORKER_STATUS_METRIC,
  workerTypeLabel,
} from '@/constants/taskCenter';
import { queryTaskCenterSummary, type FailuresSummary } from '@/services/taskCenter';
import {
  type LeasedTaskRow,
  type WorkerMonitorData,
  type WorkerMonitorInstance,
  type WorkerMonitorSummary,
  getWorkersMonitor,
} from '@/services/workers';

const POLL_MS = 5000;

const EMPTY_SUMMARY: WorkerMonitorSummary = { running: 0, stale: 0, stopped: 0 };

const LEASE_SECTIONS: {
  title: string;
  dataKey: keyof WorkerMonitorData['leasedTasks'];
}[] = [
  { title: 'Collection', dataKey: 'collect' },
  { title: 'AI image processing', dataKey: 'image' },
  { title: 'Order sync', dataKey: 'orderSync' },
  { title: 'Customer-message sync', dataKey: 'customerMessageSync' },
  { title: 'Product publishing', dataKey: 'productPublish' },
  { title: 'Inventory sync', dataKey: 'inventorySync' },
];


function statusTag(eff: string | undefined, raw: string) {
  const v = (eff || raw || '').trim().toLowerCase();
  const m = WORKER_EFFECTIVE_STATUS[v];
  if (!m) return <Tag>{raw || '—'}</Tag>;
  return <Tag color={m.color}>{m.text}</Tag>;
}

function WorkerStatusMetrics({ summary }: { summary: WorkerMonitorSummary }) {
  return (
    <Row gutter={[8, 8]}>
      {(['running', 'stale', 'stopped'] as const).map((key) => {
        const meta = WORKER_STATUS_METRIC[key];
        return (
          <Col xs={8} key={key}>
            <Statistic
              title={meta.text}
              value={summary[key] ?? 0}
              valueStyle={{ fontSize: 22, fontWeight: 600, color: meta.valueStyle }}
            />
          </Col>
        );
      })}
    </Row>
  );
}

export default function WorkersMonitorPage() {
  const [data, setData] = useState<WorkerMonitorData | null>(null);
  const [failSum, setFailSum] = useState<FailuresSummary | null>(null);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setInterval> | undefined;

    const tick = async () => {
      try {
        const d = await getWorkersMonitor();
        if (!cancelled) setData(d);
      } catch {
        /* keep last snapshot */
      }
      try {
        const s = await queryTaskCenterSummary();
        if (!cancelled) setFailSum(s);
      } catch {
        /* ignore */
      }
    };

    const arm = () => {
      if (timer) clearInterval(timer);
      timer = undefined;
      if (typeof document !== 'undefined' && document.visibilityState !== 'hidden') {
        timer = setInterval(tick, POLL_MS);
      }
    };

    void tick();
    arm();

    const onVis = () => {
      if (typeof document !== 'undefined' && document.visibilityState !== 'hidden') {
        void tick();
      }
      arm();
    };

    document.addEventListener('visibilitychange', onVis);
    return () => {
      cancelled = true;
      if (timer) clearInterval(timer);
      document.removeEventListener('visibilitychange', onVis);
    };
  }, []);

  const columns: ProColumns<WorkerMonitorInstance>[] = useMemo(
    () => [
      {
        title: 'Type',
        dataIndex: 'workerType',
        width: 120,
        render: (_, row) => workerTypeLabel(row.workerType),
      },
      {
        title: 'Worker ID',
        dataIndex: 'workerId',
        ellipsis: true,
        copyable: true,
      },
      {
        title: 'Host',
        dataIndex: 'hostname',
        width: 140,
        ellipsis: true,
      },
      {
        title: 'System PID',
        dataIndex: 'pid',
        width: 80,
      },
      {
        title: 'Status',
        dataIndex: 'status',
        width: 110,
        render: (_, row) => statusTag(row.effectiveStatus, row.status),
      },
      {
        title: 'Last heartbeat',
        dataIndex: 'lastHeartbeatAt',
        width: 172,
        render: (_, row) => formatDateTime(row.lastHeartbeatAt),
      },
      {
        title: 'Started',
        dataIndex: 'startedAt',
        width: 172,
        render: (_, row) => formatDateTime(row.startedAt),
      },
    ],
    [],
  );

  const leaseCols = (): ProColumns<LeasedTaskRow>[] => [
    { title: 'Task ID', dataIndex: 'id', copyable: true, ellipsis: true },
    { title: 'Status', dataIndex: 'status', width: 90, render: (_, row) => commonStatusLabel(row.status) },
    { title: 'Locked by', dataIndex: 'lockedBy', ellipsis: true },
    {
      title: 'Lease expires',
      dataIndex: 'lockedUntil',
      width: 172,
      render: (_, r) => formatDateTime(r.lockedUntil || undefined),
    },
    {
      title: 'Updated',
      dataIndex: 'updatedAt',
      width: 172,
      render: (_, r) => formatDateTime(r.updatedAt),
    },
  ];

  const bt = data?.byType ?? {};
  const summary = data?.summary ?? EMPTY_SUMMARY;

  return (
    <TmPageContainer
      title="Worker monitor"
      subTitle="Check that collection, image processing, order sync, product publishing, and other background tasks are running normally."
    >
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
        Shows the process status and active work for each worker type. Refreshes every {POLL_MS / 1000} seconds and pauses while this page is hidden.
      </Typography.Paragraph>

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} xl={14}>
          <ProCard variant="outlined" title="Failed-task center snapshot" size="small">
            {failSum ? (
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Row gutter={[16, 8]}>
                  <Col xs={12} sm={8}>
                    <Statistic title="Failed (normalized)" value={failSum.totalFailed ?? 0} />
                  </Col>
                  <Col xs={12} sm={8}>
                    <Statistic title="Retryable" value={failSum.retryableCount ?? 0} />
                  </Col>
                  <Col xs={24} sm={8}>
                    <Statistic
                      title="Retrying / stale / timed out"
                      value={`${failSum.retryingTotal ?? 0}/${failSum.staleTotal ?? 0}/${failSum.leaseExpiredTotal ?? 0}`}
                    />
                  </Col>
                </Row>
                <Row gutter={[16, 8]}>
                  <Col xs={12} sm={8}>
                    <Statistic title="Marked ignored" value={failSum.ignoredCount ?? 0} />
                  </Col>
                  <Col xs={12} sm={8}>
                    <Statistic title="Marked handled" value={failSum.handledCount ?? 0} />
                  </Col>
                  <Col xs={24} sm={8} style={{ display: 'flex', alignItems: 'flex-end' }}>
                    <Button type="primary" block onClick={() => history.push('/ops/task-center/failures')}>
                      Open failed-task center
                    </Button>
                  </Col>
                </Row>
              </Space>
            ) : (
              <Typography.Text type="secondary">Loading…</Typography.Text>
            )}
          </ProCard>
        </Col>
        <Col xs={24} xl={10}>
          <ProCard variant="outlined" title="Instance status summary" size="small">
            <WorkerStatusMetrics summary={summary} />
          </ProCard>
        </Col>
      </Row>

      <Typography.Title level={5} style={{ marginTop: 0, marginBottom: 12 }}>
        By worker type
      </Typography.Title>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        {WORKER_MONITOR_TYPE_KEYS.map((k) => {
          const typeSummary = bt[k] ?? EMPTY_SUMMARY;
          const total = (typeSummary.running ?? 0) + (typeSummary.stale ?? 0) + (typeSummary.stopped ?? 0);
          return (
            <Col xs={24} sm={12} lg={8} key={k}>
              <ProCard
                variant="outlined"
                size="small"
                title={TASK_CENTER_TASK_TYPE_LABEL[k] || k}
                extra={
                  <Tag color={total > 0 ? 'processing' : 'default'}>{total} instances</Tag>
                }
              >
                <WorkerStatusMetrics summary={typeSummary} />
              </ProCard>
            </Col>
          );
        })}
      </Row>

      <ProCard title="Worker process list" variant="outlined" style={{ marginBottom: 16 }}>
        <ProTable<WorkerMonitorInstance>
          rowKey={(r) => r.workerInstanceId || r.workerId}
          columns={columns}
          dataSource={data?.instances ?? []}
          search={false}
          options={false}
          pagination={{ pageSize: 20 }}
        />
      </ProCard>

      {LEASE_SECTIONS.map(({ title, dataKey }) => (
        <ProCard
          key={dataKey}
          title={`Leased tasks · ${title}`}
          variant="outlined"
          style={{ marginBottom: 16 }}
          size="small"
        >
          <ProTable<LeasedTaskRow>
            rowKey="id"
            columns={leaseCols()}
            dataSource={data?.leasedTasks[dataKey] ?? []}
            search={false}
            options={false}
            pagination={{ pageSize: 10 }}
          />
        </ProCard>
      ))}
    </TmPageContainer>
  );
}

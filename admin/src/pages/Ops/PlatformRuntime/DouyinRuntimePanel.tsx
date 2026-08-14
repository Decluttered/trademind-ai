import {
  ApiOutlined,
  BellOutlined,
  CheckCircleFilled,
  CloudServerOutlined,
  ExclamationCircleFilled,
  MinusCircleFilled,
  PauseCircleOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  ShopOutlined,
  StopOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { ActionBar, SectionCard, TechnicalDetails } from '@/components/ui';
import { releaseGateConclusionLabel } from '@/constants/platformRuntime';
import { confirmStoragePublicTest } from '@/constants/sensitiveActions';
import { formatDateTime } from '@/utils/formatTime';
import { history, Link } from '@umijs/max';
import {
  Alert,
  Button,
  Descriptions,
  Input,
  List,
  Modal,
  Space,
  Spin,
  Statistic,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react';
import {
  emergencyDisableDouyinRuntime,
  getDouyinHealth,
  getDouyinMetricsSummary,
  getDouyinReleaseGate,
  getDouyinRuntimeStatus,
  pauseDouyinRuntime,
  resumeDouyinRuntime,
  runDouyinHealthCheck,
  testStoragePublicAccess,
  type DouyinHealth,
  type DouyinHealthSection,
  type DouyinMetricsSummary,
  type DouyinReleaseGate,
  type DouyinReleaseGateItem,
  type DouyinRuntimeStatus,
  type HealthLayerStatus,
  type ReleaseGateStatus,
} from '@/services/douyinProduction';
import styles from './index.less';

const { Text, Paragraph } = Typography;

const HEALTH_META: Record<HealthLayerStatus, { color: string; text: string }> = {
  healthy: { color: 'success', text: '正常' },
  degraded: { color: 'warning', text: '需检查' },
  unhealthy: { color: 'error', text: '异常' },
  disabled: { color: 'default', text: '已停用' },
};

const GATE_META: Record<ReleaseGateStatus, { color: string; text: string }> = {
  passed: { color: 'success', text: '通过' },
  warning: { color: 'warning', text: '警告' },
  failed: { color: 'error', text: '未通过' },
  blocked: { color: 'default', text: '阻塞' },
  not_checked: { color: 'default', text: '未检查' },
};

function healthTag(status?: string) {
  const st = (status || '') as HealthLayerStatus;
  const m = HEALTH_META[st] ?? { color: 'default', text: status || '未知' };
  return (
    <Tag color={m.color} style={{ margin: 0 }}>
      {m.text}
    </Tag>
  );
}

function gateTag(status?: string) {
  const st = (status || '') as ReleaseGateStatus;
  const m = GATE_META[st] ?? { color: 'default', text: status || '未知' };
  return (
    <Tag color={m.color} style={{ margin: 0 }}>
      {m.text}
    </Tag>
  );
}

function runtimeStatusTag(status?: string) {
  switch (status) {
    case 'normal':
      return <Tag color="success">正常运行</Tag>;
    case 'paused':
      return <Tag color="warning">已暂停</Tag>;
    case 'emergency_disabled':
      return <Tag color="error">紧急停用</Tag>;
    default:
      return <Tag>等待数据</Tag>;
  }
}

function healthOverviewIcon(status?: HealthLayerStatus) {
  switch (status) {
    case 'healthy':
      return <CheckCircleFilled />;
    case 'degraded':
    case 'unhealthy':
      return <ExclamationCircleFilled />;
    default:
      return <MinusCircleFilled />;
  }
}

function failuresHref(params: Record<string, string>): string {
  const sp = new URLSearchParams(params);
  return `/ops/task-center/failures?${sp.toString()}`;
}

const DOUYIN_TASK_HEALTH_DETAIL_LABEL: Record<string, string> = {
  failedPending: '待处理失败任务',
  recoveryRequired: '需要人工检查',
  resultUnknown: '平台结果暂时无法确认',
  stale24h: '任务执行时间过长（24h）',
  runtimeBlocked24h: '因运行状态被拦截（24h）',
};

const DOUYIN_TASK_HEALTH_DETAIL_ORDER = [
  'failedPending',
  'recoveryRequired',
  'resultUnknown',
  'stale24h',
  'runtimeBlocked24h',
] as const;

function douyinTaskHealthDetailLabel(key: string): string {
  return DOUYIN_TASK_HEALTH_DETAIL_LABEL[key] || key;
}

type IssueRow = {
  key: string;
  label: string;
  count: number;
  href: string;
  severity: 'warning' | 'error' | 'info';
};

function buildIssues(health: DouyinHealth | null, metrics: DouyinMetricsSummary | null): IssueRow[] {
  const rows: IssueRow[] = [];
  const td = health?.tasks?.details ?? {};
  const num = (k: string) => {
    const v = td[k];
    return typeof v === 'number' ? v : Number(v) || 0;
  };

  if (num('stale24h') > 0) {
    rows.push({
      key: 'stale',
      label: '任务执行时间过长',
      count: num('stale24h'),
      href: failuresHref({ platform: 'douyin_shop', recoveryStatus: 'stale' }),
      severity: 'warning',
    });
  }
  if (num('resultUnknown') > 0) {
    rows.push({
      key: 'result_unknown',
      label: DOUYIN_TASK_HEALTH_DETAIL_LABEL.resultUnknown,
      count: num('resultUnknown'),
      href: failuresHref({ platform: 'douyin_shop', recoveryStatus: 'result_unknown' }),
      severity: 'warning',
    });
  }
  if (num('recoveryRequired') > 0) {
    rows.push({
      key: 'recovery_required',
      label: DOUYIN_TASK_HEALTH_DETAIL_LABEL.recoveryRequired,
      count: num('recoveryRequired'),
      href: failuresHref({ platform: 'douyin_shop', recoveryStatus: 'recovery_required' }),
      severity: 'error',
    });
  }
  if (num('failedPending') > 0) {
    rows.push({
      key: 'failed_pending',
      label: DOUYIN_TASK_HEALTH_DETAIL_LABEL.failedPending,
      count: num('failedPending'),
      href: failuresHref({ platform: 'douyin_shop' }),
      severity: 'error',
    });
  }
  if ((metrics?.recoveryFailedTotal ?? 0) > 0) {
    rows.push({
      key: 'recovery_failed',
      label: '恢复失败',
      count: metrics!.recoveryFailedTotal,
      href: failuresHref({ platform: 'douyin_shop', recoveryStatus: 'recovery_failed' }),
      severity: 'error',
    });
  }
  if ((metrics?.tokenRefreshFailedTotal ?? 0) >= 3) {
    rows.push({
      key: 'token_refresh',
      label: '访问令牌刷新失败（24h）',
      count: metrics!.tokenRefreshFailedTotal,
      href: '/shops/manage',
      severity: 'warning',
    });
  }
  if (health?.auth?.status === 'unhealthy') {
    rows.push({
      key: 'auth_expired',
      label: health.auth.label || '店铺授权异常',
      count: Number(td.expiredShops) || 0,
      href: '/shops/manage',
      severity: 'error',
    });
  }
  return rows;
}

function HealthSectionCard({
  title,
  icon,
  section,
  extra,
}: {
  title: string;
  icon?: ReactNode;
  section?: DouyinHealthSection;
  extra?: React.ReactNode;
}) {
  return (
    <SectionCard
      compact
      className={styles.healthCard}
      style={{ height: '100%', marginBlockEnd: 0 }}
      title={
        <Space size={8}>
          <span className={styles.healthIcon}>{icon}</span>
          {title}
        </Space>
      }
      headerExtra={section ? healthTag(section.status) : undefined}
    >
      {section ? (
        <Space direction="vertical" size={10} className={styles.healthCardBody}>
          <Text className={styles.healthSummary}>{section.label}</Text>
          {extra}
          {section.details && Object.keys(section.details).length > 0 ? (
            <TechnicalDetails>
              <pre style={{ margin: 0, fontSize: 12, whiteSpace: 'pre-wrap' }}>
                {JSON.stringify(section.details, null, 2)}
              </pre>
            </TechnicalDetails>
          ) : null}
        </Space>
      ) : (
        <Text type="secondary">暂无数据</Text>
      )}
    </SectionCard>
  );
}

function MetricsGrid({ metrics }: { metrics: DouyinMetricsSummary | null }) {
  if (!metrics) return <Text type="secondary">正在载入指标...</Text>;
  const items = [
    { title: '接口请求', value: metrics.apiRequestsTotal, tone: 'default' },
    {
      title: '接口成功率',
      value: `${metrics.apiSuccessRate.toFixed(1)}%`,
      tone: 'success',
    },
    {
      title: '停滞标记',
      value: metrics.staleTasksTotal,
      tone: metrics.staleTasksTotal > 0 ? 'warning' : 'default',
    },
    { title: '恢复成功', value: metrics.recoverySuccessTotal, tone: 'success' },
    {
      title: '恢复失败',
      value: metrics.recoveryFailedTotal,
      tone: metrics.recoveryFailedTotal > 0 ? 'danger' : 'default',
    },
    {
      title: '草稿创建',
      value: metrics.productDraftCreateTotal,
      tone: 'default',
    },
    {
      title: '草稿失败',
      value: metrics.productDraftCreateFailedTotal,
      tone: metrics.productDraftCreateFailedTotal > 0 ? 'danger' : 'default',
    },
    { title: '订单拉取', value: metrics.orderFetchedTotal, tone: 'default' },
    {
      title: '库存同步成功',
      value: metrics.inventorySyncSuccessTotal,
      tone: 'success',
    },
    {
      title: '库存同步失败',
      value: metrics.inventorySyncFailedTotal,
      tone: metrics.inventorySyncFailedTotal > 0 ? 'danger' : 'default',
    },
    {
      title: '待处理失败',
      value: metrics.failureTasksPending,
      tone: metrics.failureTasksPending > 0 ? 'danger' : 'default',
    },
    {
      title: '限流次数',
      value: metrics.apiRateLimitedTotal,
      tone: metrics.apiRateLimitedTotal > 0 ? 'warning' : 'default',
    },
  ];
  return (
    <div className={styles.metricsGrid}>
      {items.map((it) => (
        <div className={styles.metricItem} data-tone={it.tone} key={it.title}>
          <Statistic title={it.title} value={it.value} valueStyle={{ fontSize: 20 }} />
        </div>
      ))}
    </div>
  );
}

function ReleaseGatePanel({ gate }: { gate: DouyinReleaseGate | null }) {
  if (!gate) return <Text type="secondary">正在载入门禁清单...</Text>;
  return (
    <Space direction="vertical" size={12} className={styles.releaseGatePanel}>
      <Space wrap className={styles.releaseGateSummary}>
        <Text strong>发布结论</Text>
        <Tag color="processing">{releaseGateConclusionLabel(gate.overallConclusion)}</Tag>
        <Text type="secondary">检查时间：{formatDateTime(gate.checkedAt)}</Text>
      </Space>
      <List<DouyinReleaseGateItem>
        size="small"
        className={styles.releaseGateList}
        dataSource={gate.items}
        renderItem={(item) => (
          <List.Item>
            <div className={styles.releaseGateRow}>
              <Space size={8}>
                <Text>{item.label}</Text>
                {gateTag(item.status)}
              </Space>
              {item.message ? <Text type="secondary">{item.message}</Text> : null}
            </div>
          </List.Item>
        )}
      />
    </Space>
  );
}

export default function DouyinRuntimePanel() {
  const [loading, setLoading] = useState(true);
  const [healthChecking, setHealthChecking] = useState(false);
  const [storageTesting, setStorageTesting] = useState(false);
  const [health, setHealth] = useState<DouyinHealth | null>(null);
  const [metrics, setMetrics] = useState<DouyinMetricsSummary | null>(null);
  const [gate, setGate] = useState<DouyinReleaseGate | null>(null);
  const [runtime, setRuntime] = useState<DouyinRuntimeStatus | null>(null);
  const [reason, setReason] = useState('');
  const [loadError, setLoadError] = useState(false);

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      const [h, m, g, rt] = await Promise.all([
        getDouyinHealth(),
        getDouyinMetricsSummary(),
        getDouyinReleaseGate(),
        getDouyinRuntimeStatus(),
      ]);
      setHealth(h);
      setMetrics(m);
      setGate(g);
      setRuntime(rt);
      setLoadError(false);
    } catch (e: unknown) {
      setLoadError(true);
      message.error((e as Error)?.message || '加载运行状态失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  const issues = useMemo(() => buildIssues(health, metrics), [health, metrics]);

  const confirmRuntimeChange = (title: string, onOk: () => Promise<void>) => {
    if (!reason.trim()) {
      message.warning('请填写变更原因');
      return;
    }
    Modal.confirm({
      title,
      content: (
        <div>
          <Paragraph type="secondary">此操作将影响抖店任务执行，请确认原因已记录。</Paragraph>
          <Text>原因：{reason}</Text>
        </div>
      ),
      okText: '确认',
      cancelText: '取消',
      onOk,
    });
  };

  const runHealthCheck = async () => {
    setHealthChecking(true);
    try {
      const h = await runDouyinHealthCheck();
      setHealth(h);
      message.success('健康检查完成，已触发告警扫描');
      const g = await getDouyinReleaseGate();
      setGate(g);
    } catch (e: unknown) {
      message.error((e as Error)?.message || '健康检查失败');
    } finally {
      setHealthChecking(false);
    }
  };

  const testStorage = () => {
    confirmStoragePublicTest(async () => {
      setStorageTesting(true);
      try {
        const res = await testStoragePublicAccess();
        if (res.ok) {
          message.success(res.message || '存储公网访问正常');
        } else {
          message.warning(res.message || '存储公网访问需检查');
        }
        const h = await getDouyinHealth();
        setHealth(h);
      } catch (e: unknown) {
        message.error((e as Error)?.message || '存储公网访问测试失败');
      } finally {
        setStorageTesting(false);
      }
    });
  };

  const gray = health?.grayRelease;
  const runtimeStatus = runtime?.status ?? health?.runtime?.status;
  const lastChangeReason = runtime?.reason;
  const lastChangedAt = runtime?.changedAt;

  return (
    <Spin spinning={loading}>
      <div className={styles.panelContent}>
        {loadError ? (
          <Alert
            showIcon
            type="error"
            message="运行状态加载失败"
            description="已保留当前页面内容，请检查后端服务后重试。"
            action={
              <Button size="small" onClick={() => void loadAll()}>
                重新加载
              </Button>
            }
          />
        ) : null}

        <section className={styles.statusOverview} data-status={health?.overallStatus ?? 'unknown'} aria-label="抖店运行概览">
          <div className={styles.statusMain}>
            <span className={styles.statusIcon} data-status={health?.overallStatus ?? 'unknown'}>
              {healthOverviewIcon(health?.overallStatus)}
            </span>
            <div className={styles.statusCopy}>
              <Text type="secondary" className={styles.statusEyebrow}>
                抖店运行概览
              </Text>
              <div className={styles.statusTitleRow}>
                <Text strong className={styles.statusTitle}>
                  {health?.overallLabel || '等待状态数据'}
                </Text>
                {healthTag(health?.overallStatus)}
              </div>
              <Text type="secondary">{health?.checkedAt ? `最近检查：${formatDateTime(health.checkedAt)}` : '尚未完成健康检查'}</Text>
            </div>
          </div>
          <div className={styles.summaryActions}>
            <Space wrap size={8}>
              <Button type="primary" icon={<ReloadOutlined />} loading={healthChecking} onClick={() => void runHealthCheck()}>
                运行健康检查
              </Button>
              <Tooltip title="刷新全部状态">
                <Button aria-label="刷新全部状态" icon={<ReloadOutlined />} onClick={() => void loadAll()} loading={loading} />
              </Tooltip>
              <Button icon={<BellOutlined />} onClick={() => history.push('/ops/task-center/alerts')}>
                告警中心
              </Button>
              <Button icon={<WarningOutlined />} onClick={() => history.push('/ops/task-center/failures?platform=douyin_shop')}>
                失败任务
              </Button>
              <Link to="/settings/platforms">
                <Button icon={<SettingOutlined />}>平台接入设置</Button>
              </Link>
            </Space>
          </div>
          {gray?.enabled ? (
            <Alert
              className={styles.grayReleaseAlert}
              showIcon
              type="info"
              message="灰度发布已启用"
              description={
                <Space direction="vertical" size={4}>
                  <Text>
                    写操作：{gray.writeOperationsEnabled ? '已开启' : '已关闭'} · 定时订单同步：
                    {gray.scheduledOrderSyncEnabled ? '已开启' : '已关闭'} · 定时库存同步：
                    {gray.scheduledInventorySyncEnabled ? '已开启' : '已关闭'}
                  </Text>
                  {gray.shopIds?.length ? <Text type="secondary">灰度店铺：{gray.shopIds.join(', ')}</Text> : <Text type="secondary">未配置灰度店铺列表</Text>}
                </Space>
              }
            />
          ) : null}
        </section>

        <SectionCard title="运行控制" description="暂停或紧急停用后，后台任务进程将不再调用抖店写接口。" className={styles.controlCard}>
          <div className={styles.controlGrid}>
            <div className={styles.controlStatus} data-status={runtimeStatus ?? 'unknown'}>
              <Text type="secondary">当前状态</Text>
              <div className={styles.controlStatusValue}>
                {runtimeStatusTag(runtimeStatus)}
                {runtime?.message ? <Text>{runtime.message}</Text> : null}
              </div>
              {lastChangeReason ? <Text type="secondary">最近变更原因：{lastChangeReason}</Text> : null}
              {lastChangedAt ? <Text type="secondary">变更时间：{formatDateTime(lastChangedAt)}</Text> : null}
            </div>
            <div className={styles.controlForm}>
              <div>
                <Text strong>变更原因</Text>
                <Text type="secondary" className={styles.controlHint}>
                  原因将随运行状态变更记录，提交前请确认影响范围。
                </Text>
              </div>
              <Input.TextArea rows={3} placeholder="填写本次操作原因（必填）" value={reason} onChange={(e) => setReason(e.target.value)} />
              <ActionBar className={styles.controlActions}>
                <Button
                  icon={<PauseCircleOutlined />}
                  onClick={() =>
                    confirmRuntimeChange('暂停抖店任务？', async () => {
                      const res = await pauseDouyinRuntime(reason.trim());
                      setRuntime(res);
                      setReason('');
                      message.success('已暂停');
                      void loadAll();
                    })
                  }
                >
                  暂停任务
                </Button>
                <Button
                  type="primary"
                  icon={<PlayCircleOutlined />}
                  onClick={() =>
                    confirmRuntimeChange('恢复抖店运行？', async () => {
                      const res = await resumeDouyinRuntime(reason.trim());
                      setRuntime(res);
                      setReason('');
                      message.success('已恢复运行');
                      void loadAll();
                    })
                  }
                >
                  恢复运行
                </Button>
                <Button
                  danger
                  icon={<StopOutlined />}
                  onClick={() =>
                    confirmRuntimeChange('紧急停用抖店？', async () => {
                      const res = await emergencyDisableDouyinRuntime(reason.trim());
                      setRuntime(res);
                      setReason('');
                      message.warning('已紧急停用');
                      void loadAll();
                    })
                  }
                >
                  紧急停用
                </Button>
              </ActionBar>
            </div>
          </div>
        </SectionCard>

        <div className={styles.healthGrid}>
          <HealthSectionCard
            title="应用配置"
            icon={<ApiOutlined />}
            section={health?.config}
            extra={
              <Link to="/settings/platforms">
                <Button size="small" type="link">
                  前往平台接入设置
                </Button>
              </Link>
            }
          />
          <HealthSectionCard
            title="店铺授权"
            icon={<ShopOutlined />}
            section={health?.auth}
            extra={
              <Link to="/shops/manage">
                <Button size="small" type="link">
                  前往店铺管理
                </Button>
              </Link>
            }
          />
          <HealthSectionCard
            title="存储公网访问"
            icon={<CloudServerOutlined />}
            section={health?.storage}
            extra={
              <Space wrap>
                <Button size="small" loading={storageTesting} onClick={() => void testStorage()}>
                  测试公网访问
                </Button>
                <Link to="/settings/storage">
                  <Button size="small" type="link">
                    存储设置
                  </Button>
                </Link>
              </Space>
            }
          />
          <HealthSectionCard title="接口调用" icon={<SafetyCertificateOutlined />} section={health?.api} />
        </div>

        <SectionCard title="24 小时指标" description="滚动 24 小时窗口内的抖店运行指标。">
          {metrics?.generatedAt ? (
            <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
              统计时间：{formatDateTime(metrics.generatedAt)}
            </Text>
          ) : null}
          <MetricsGrid metrics={metrics} />
        </SectionCard>

        <div className={styles.lowerGrid}>
          <SectionCard title="待处理问题" description="点击条目跳转到对应处理页面。" className={styles.lowerCard}>
            {issues.length ? (
              <List<IssueRow>
                size="small"
                className={styles.issueList}
                dataSource={issues}
                renderItem={(item) => (
                  <List.Item
                    actions={[
                      <Button key="go" type="link" size="small" onClick={() => history.push(item.href)}>
                        去处理
                      </Button>,
                    ]}
                  >
                    <Space>
                      <Tag color={item.severity === 'error' ? 'error' : item.severity === 'warning' ? 'warning' : 'processing'}>{item.count}</Tag>
                      <Text>{item.label}</Text>
                    </Space>
                  </List.Item>
                )}
              />
            ) : (
              <Alert showIcon type="success" message="暂无需要立即处理的问题" />
            )}
          </SectionCard>

          <SectionCard title="发布门禁清单" description="发布候选检查项；真实端到端联调仍可能因缺少真实凭证而阻塞。" className={styles.lowerCard}>
            <ReleaseGatePanel gate={gate} />
          </SectionCard>
        </div>

        {health?.tasks ? (
          <SectionCard title="任务健康摘要">
            <Descriptions size="small" column={{ xs: 1, sm: 2, md: 3 }}>
              {(() => {
                const details = health.tasks.details ?? {};
                const ordered = DOUYIN_TASK_HEALTH_DETAIL_ORDER.filter((k) => details[k] !== undefined && details[k] !== null);
                const extra = Object.keys(details).filter((k) => !(DOUYIN_TASK_HEALTH_DETAIL_ORDER as readonly string[]).includes(k));
                return [...ordered, ...extra].map((k) => (
                  <Descriptions.Item key={k} label={douyinTaskHealthDetailLabel(k)}>
                    {String(details[k])}
                  </Descriptions.Item>
                ));
              })()}
            </Descriptions>
          </SectionCard>
        ) : null}
      </div>
    </Spin>
  );
}

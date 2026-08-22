import {
  BellOutlined,
  CheckCircleOutlined,
  CloudServerOutlined,
  ExclamationCircleOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import { history } from '@umijs/max';
import { Button, Descriptions, Spin, Tag, Tooltip, Typography, message } from 'antd';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ErrorAlert, MetricCard, SectionCard, TmPageContainer } from '@/components/ui';
import {
  fetchObservabilityOverview,
  type ObservabilityOverview,
} from '@/services/observability';
import { formatDateTime } from '@/utils/formatTime';
import styles from './index.less';

const { Text } = Typography;

type StatusMeta = {
  color: 'default' | 'processing' | 'success' | 'error' | 'warning';
  text: string;
  intent: 'default' | 'success' | 'warning' | 'danger' | 'data';
};

const STATUS_META: Record<string, StatusMeta> = {
  healthy: { color: 'success', text: 'Healthy', intent: 'success' },
  needs_attention: { color: 'error', text: 'Needs attention', intent: 'danger' },
  waiting: { color: 'processing', text: 'Waiting for first evaluation', intent: 'warning' },
  warming_up: { color: 'processing', text: 'Evaluator warming up', intent: 'warning' },
  succeeded: { color: 'success', text: 'Latest evaluation succeeded', intent: 'success' },
  failed: { color: 'error', text: 'Latest evaluation failed', intent: 'danger' },
  disabled: { color: 'default', text: 'Disabled', intent: 'default' },
  active: { color: 'success', text: 'Active', intent: 'success' },
  unavailable: { color: 'error', text: 'Unavailable', intent: 'danger' },
  unprotected: { color: 'error', text: 'Protection incomplete', intent: 'danger' },
  achieved: { color: 'success', text: 'Within objective', intent: 'success' },
  violated: { color: 'error', text: 'Outside objective', intent: 'danger' },
  insufficient_data: { color: 'warning', text: 'Insufficient data', intent: 'warning' },
  standard_protocol_ready: { color: 'success', text: 'Latest export succeeded', intent: 'success' },
  export_pending: { color: 'processing', text: 'Waiting for first export', intent: 'warning' },
  real_backend_deferred: { color: 'default', text: 'Export backend not configured', intent: 'default' },
  export_degraded: { color: 'error', text: 'Export degraded', intent: 'danger' },
  incomplete: { color: 'error', text: 'Configuration incomplete', intent: 'danger' },
};

function statusMeta(status?: string): StatusMeta {
  return STATUS_META[status ?? ''] ?? { color: 'default', text: status || 'Unknown', intent: 'default' };
}

function StatusValue({ status }: { status?: string }) {
  const meta = statusMeta(status);
  return <Tag color={meta.color}>{meta.text}</Tag>;
}

export default function ObservabilityCenterPage() {
  const [overview, setOverview] = useState<ObservabilityOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const result = await fetchObservabilityOverview();
      setOverview(result?.data ?? null);
      setLoadError(false);
    } catch {
      message.error('Observability data could not be loaded.');
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const overallMeta = statusMeta(overview?.overallStatus);
  const evaluationMeta = statusMeta(overview?.evaluation?.status);
  const metricMeta = statusMeta(overview?.metrics?.status);
  const alertMeta = statusMeta(overview?.alerts?.status);
  const activeAlerts = overview?.alerts?.active ?? 0;
  const alertValue = overview?.alerts?.status === 'active' ? activeAlerts : alertMeta.text;
  const alertDescription = overview?.alerts?.status === 'active'
    ? `Critical ${overview?.alerts?.critical ?? 0} · Warning ${overview?.alerts?.warning ?? 0}`
    : overview?.alerts?.status === 'disabled' ? 'System alerts are disabled' : 'System alerts cannot be read';
  const alertIntent = overview?.alerts?.status === 'active'
    ? activeAlerts > 0 ? 'danger' : 'success'
    : alertMeta.intent;
  const lastUpdated = useMemo(() => formatDateTime(overview?.timestamp), [overview?.timestamp]);

  return (
    <TmPageContainer
      title="Observability"
      subTitle="View system metrics, alert evaluation, SLOs, and telemetry export status."
      className={styles.page}
      extra={[
        <Button
          key="alerts"
          icon={<BellOutlined />}
          onClick={() => history.push('/ops/task-center/alerts?source=system')}
        >
          View system alerts
        </Button>,
        <Tooltip key="reload" title="Refresh runtime status">
          <Button
            aria-label="Refresh runtime status"
            icon={<ReloadOutlined />}
            onClick={() => void load()}
            loading={loading}
          />
        </Tooltip>,
      ]}
    >
      {loadError ? (
        <ErrorAlert
          title="Observability overview could not be loaded"
          actionHint="Check backend health, then try again."
          className={styles.error}
        />
      ) : null}

      {!overview && loading ? (
        <div className={styles.loading} aria-label="Loading observability overview">
          <Spin size="large" />
        </div>
      ) : null}

      {overview ? <Spin spinning={loading}>
        <div className={styles.content}>
          <div className={styles.metricsGrid}>
            <MetricCard
              title="Overall runtime status"
              value={overallMeta.text}
              description={`Environment: ${overview?.environment || 'Unknown'}`}
              icon={overview?.overallStatus === 'healthy' ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}
              intent={overallMeta.intent}
            />
            <MetricCard
              title="Active system alerts"
              value={alertValue}
              description={alertDescription}
              icon={<BellOutlined />}
              intent={alertIntent}
              onClick={() => history.push('/ops/task-center/alerts?source=system')}
            />
            <MetricCard
              title="Alert evaluator"
              value={evaluationMeta.text}
              description={`${overview?.evaluation?.rulesChecked ?? 0} rules checked`}
              icon={<SafetyCertificateOutlined />}
              intent={evaluationMeta.intent}
            />
            <MetricCard
              title="Metrics registry"
              value={metricMeta.text}
              description={overview?.metrics?.internalOnly ? 'Internal endpoint is protected' : 'Endpoint is not restricted to internal access'}
              icon={<CloudServerOutlined />}
              intent={metricMeta.intent}
            />
          </div>

          <div className={styles.detailGrid}>
            <SectionCard title="Alerts and service objectives" variant="outlined" compact>
              <div className={styles.statusList}>
                <div className={styles.statusRow}>
                  <div>
                    <Text strong>Alert rule evaluation</Text>
                    <Text type="secondary">Window metrics, sample protection, and recovery detection</Text>
                  </div>
                  <StatusValue status={overview?.evaluation?.status} />
                </div>
                <div className={styles.statusRow}>
                  <div>
                    <Text strong>SLO evaluation</Text>
                    <Text type="secondary">Availability, latency, and critical background-task objectives</Text>
                  </div>
                  <StatusValue status={overview?.slo?.status} />
                </div>
              </div>
              <Descriptions className={styles.descriptions} column={{ xs: 1, sm: 2 }} size="small">
                <Descriptions.Item label="Latest alert evaluation">
                  {formatDateTime(overview?.evaluation?.lastEvaluatedAt)}
                </Descriptions.Item>
                <Descriptions.Item label="Latest SLO evaluation">
                  {formatDateTime(overview?.slo?.lastEvaluatedAt)}
                </Descriptions.Item>
                <Descriptions.Item label="Triggered this run">
                  {overview?.evaluation?.alertsFired ?? 0}
                </Descriptions.Item>
                <Descriptions.Item label="Recovered this run">
                  {overview?.evaluation?.alertsResolved ?? 0}
                </Descriptions.Item>
              </Descriptions>
            </SectionCard>

            <SectionCard title="Metrics and telemetry" variant="outlined" compact>
              <div className={styles.statusList}>
                <div className={styles.statusRow}>
                  <div>
                    <Text strong>Prometheus metrics</Text>
                    <Text type="secondary" copyable={{ text: overview?.metrics?.path || '' }}>
                      {overview?.metrics?.path || 'Path not configured'}
                    </Text>
                  </div>
                  <StatusValue status={overview?.metrics?.status} />
                </div>
                <div className={styles.statusRow}>
                  <div>
                    <Text strong>Telemetry export</Text>
                    <Text type="secondary">{overview?.telemetry?.protocol || 'Protocol not configured'}</Text>
                  </div>
                  <StatusValue status={overview?.telemetry?.status} />
                </div>
              </div>
              <Descriptions className={styles.descriptions} column={{ xs: 1, sm: 2 }} size="small">
                <Descriptions.Item label="Successful exports">
                  {overview?.telemetry?.exportSuccess ?? 0}
                </Descriptions.Item>
                <Descriptions.Item label="Failed exports">
                  {overview?.telemetry?.exportFailures ?? 0}
                </Descriptions.Item>
                <Descriptions.Item label="Failures dropped">
                  {overview?.telemetry?.dropped ?? 0}
                </Descriptions.Item>
              </Descriptions>
            </SectionCard>
          </div>

          <div className={styles.runtimeBand}>
            <div className={styles.runtimeItems}>
              <span><Text type="secondary">Mode</Text><Text strong>{overview?.mode || 'Unknown'}</Text></span>
              <span><Text type="secondary">Environment</Text><Text strong>{overview?.environment || 'Unknown'}</Text></span>
              <span><Text type="secondary">Metrics endpoint</Text><Text strong>{overview?.metrics?.internalOnly ? 'Internal only' : 'Unprotected'}</Text></span>
              <span><Text type="secondary">Last refreshed</Text><Text strong>{lastUpdated}</Text></span>
            </div>
            <StatusValue status={overview?.telemetry?.status} />
          </div>
        </div>
      </Spin> : null}
    </TmPageContainer>
  );
}

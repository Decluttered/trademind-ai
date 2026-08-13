import { ReloadOutlined } from '@ant-design/icons';
import { history } from '@umijs/max';
import { Button, Card, Descriptions, Space, Spin, Tag, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { ErrorAlert, TmPageContainer } from '@/components/ui';
import {
  fetchObservabilityOverview,
  type ObservabilityOverview,
} from '@/services/observability';

function otlpStatusMeta(status?: string) {
  switch (status) {
    case 'standard_protocol_ready':
      return { color: 'green', text: '标准协议就绪' };
    case 'real_backend_deferred':
      return { color: 'gold', text: '未配置导出后端' };
    case 'export_degraded':
      return { color: 'orange', text: '导出降级' };
    case 'disabled':
      return { color: 'default', text: '未启用' };
    case 'incomplete':
      return { color: 'red', text: '未完成' };
    default:
      return { color: 'default', text: status || '-' };
  }
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
      message.error('加载可观测性数据失败');
      setOverview(null);
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const exporterMeta = otlpStatusMeta(overview?.runtimeStatus?.otlpExporter);

  return (
    <TmPageContainer
      title="可观测性中心"
      extra={[
        <Button key="alerts" onClick={() => history.push('/ops/task-center/alerts?source=system')}>
          查看系统告警
        </Button>,
        <Button
          key="reload"
          icon={<ReloadOutlined />}
          onClick={() => void load()}
          loading={loading}
        >
          刷新
        </Button>,
      ]}
    >
      {loadError ? (
        <ErrorAlert
          title="可观测性概览加载失败"
          actionHint="请检查后端健康状态后重试。"
          style={{ marginBottom: 16 }}
        />
      ) : null}
      <Spin spinning={loading}>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Card title="系统概览">
            <Descriptions column={{ xs: 1, sm: 2 }} size="small">
              <Descriptions.Item label="模式">{overview?.mode ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="环境">{overview?.environment ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="指标">
                {overview?.metricsEnabled ? '已启用' : '未启用'}
              </Descriptions.Item>
              <Descriptions.Item label="链路追踪">
                {overview?.tracingEnabled ? '已启用' : '未启用'}
              </Descriptions.Item>
              <Descriptions.Item label="指标路径">{overview?.metricsPath ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="内部保护">
                {overview?.metricsInternal ? '是' : '否'}
              </Descriptions.Item>
              <Descriptions.Item label="遥测导出">
                <Tag color={exporterMeta.color}>{exporterMeta.text}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="协议">
                {overview?.runtimeStatus?.otlpProtocol ?? '-'}
              </Descriptions.Item>
              <Descriptions.Item label="导出结果">
                成功 {overview?.telemetry?.exportSuccess ?? 0} / 失败{' '}
                {overview?.telemetry?.exportFailures ?? 0} / 丢弃 {overview?.telemetry?.dropped ?? 0}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Space>
      </Spin>
    </TmPageContainer>
  );
}

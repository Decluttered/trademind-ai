import { MetricCard, SectionCard, TmPageContainer } from '@/components/ui';
import { usePermission } from '@/hooks/usePermission';
import {
  completeOfflineP10OAuth,
  createOfflineP10Credential,
  createP10IdempotencyKey,
  createP10InventoryReadRun,
  getP10Status,
  listP10Credentials,
  pauseP10Gray,
  rerunP10InventoryRead,
  revokeP10Credential,
  rotateP10Credential,
  saveP10GrayDraft,
  startOfflineP10OAuth,
  stopP10Gray,
  updateP10Allowlist,
  updateP10KillSwitches,
  type P10CredentialMetadata,
  type P10RuntimeStatus,
} from '@/services/p10Readiness';
import { PERMISSIONS } from '@/utils/permission';
import { formatDateTime } from '@/utils/formatTime';
import {
  ApiOutlined,
  KeyOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  StopOutlined,
} from '@ant-design/icons';
import {
  Alert,
  Button,
  Col,
  Descriptions,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';

type KillSwitchValues = {
  providerKillActive: boolean;
  tenantKillActive: boolean;
  shopKillActive: boolean;
  readKillActive: boolean;
};

type ShopScopeValues = { shopId: string; enabled: boolean };
type GrayValues = { shopId: string; maxSku: number };
type ShopValues = { shopId: string };
type OAuthValues = ShopValues & { redirectUri: string };

function yesNo(value: boolean) {
  return <Tag color={value ? 'green' : 'default'}>{value ? '是' : '否'}</Tag>;
}

function blockedTag(enabled: boolean) {
  return <Tag color={enabled ? 'red' : 'green'}>{enabled ? '已阻断' : '未阻断'}</Tag>;
}

function statusTag(status?: string) {
  const normalized = (status || '未配置').toLowerCase();
  const color = ['active', 'approved', 'succeeded'].includes(normalized)
    ? 'green'
    : ['failed', 'revoked', 'stopped'].includes(normalized)
      ? 'red'
      : ['pending_approval', 'paused', 'draft'].includes(normalized)
        ? 'gold'
        : 'default';
  return <Tag color={color}>{status || '未配置'}</Tag>;
}

function actionError(error: unknown) {
  return error instanceof Error && error.message ? error.message : '操作失败，请刷新后重试';
}

export default function P10ReadinessPage() {
  const { can } = usePermission();
  const canManageCredential = can(PERMISSIONS.P10_CREDENTIAL_MANAGE);
  const canManageControl = can(PERMISSIONS.P10_CONTROL_MANAGE);
  const [killForm] = Form.useForm<KillSwitchValues>();
  const [allowlistForm] = Form.useForm<ShopScopeValues>();
  const [grayForm] = Form.useForm<GrayValues>();
  const [credentialForm] = Form.useForm<ShopValues>();
  const [oauthForm] = Form.useForm<OAuthValues>();
  const [runForm] = Form.useForm<ShopValues>();
  const [status, setStatus] = useState<P10RuntimeStatus>();
  const [credentials, setCredentials] = useState<P10CredentialMetadata[]>([]);
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState('');
  const [error, setError] = useState('');
  const [offlineState, setOfflineState] = useState('');
  const requestSequence = useRef(0);

  const load = useCallback(async () => {
    const sequence = requestSequence.current + 1;
    requestSequence.current = sequence;
    setLoading(true);
    setError('');
    try {
      const [runtime, credentialItems] = await Promise.all([getP10Status(), listP10Credentials()]);
      if (requestSequence.current !== sequence) return;
      setStatus(runtime);
      setCredentials(credentialItems);
      killForm.setFieldsValue({
        providerKillActive: runtime.control.providerKillActive,
        tenantKillActive: runtime.control.tenantKillActive,
        shopKillActive: runtime.control.shopKillActive,
        readKillActive: runtime.control.readKillActive,
      });
      allowlistForm.setFieldsValue({
        shopId: runtime.allowlist?.shopId,
        enabled: runtime.allowlist?.enabled ?? false,
      });
      grayForm.setFieldsValue({ shopId: runtime.gray?.shopId, maxSku: runtime.gray?.maxSku ?? 100 });
    } catch (nextError) {
      if (requestSequence.current !== sequence) return;
      setError(actionError(nextError));
      setStatus(undefined);
      setCredentials([]);
    } finally {
      if (requestSequence.current === sequence) setLoading(false);
    }
  }, [allowlistForm, grayForm, killForm]);

  useEffect(() => {
    void load();
  }, [load]);

  const runAction = async (key: string, task: () => Promise<unknown>, success: string) => {
    setAction(key);
    setError('');
    try {
      await task();
      message.success(success);
      await load();
    } catch (nextError) {
      const nextMessage = actionError(nextError);
      setError(nextMessage);
      message.error(nextMessage);
    } finally {
      setAction('');
    }
  };

  const confirmSwitches = (values: KillSwitchValues) => {
    if (!status) return;
    Modal.confirm({
      title: '保存 Kill Switch 状态',
      content: '关闭任一阻断开关不会放开 L0 边界；Write Kill Switch 始终保持阻断。',
      okText: '保存',
      cancelText: '取消',
      onOk: () =>
        runAction(
          'switches',
          () => updateP10KillSwitches({ ...values, expectedRevision: status.control.revision }),
          'Kill Switch 状态已保存',
        ),
    });
  };

  const saveAllowlist = (values: ShopScopeValues) => {
    if (!status) return Promise.resolve();
    return runAction(
      'allowlist',
      () => updateP10Allowlist({ ...values, expectedRevision: status.allowlist?.revision ?? 0 }),
      'Allowlist 草案已保存',
    );
  };

  const saveGray = (values: GrayValues) => {
    if (!status) return Promise.resolve();
    return runAction(
      'gray',
      () => saveP10GrayDraft({ ...values, expectedRevision: status.gray?.revision ?? 0 }),
      'Gray 草案已保存，人工批准状态已重置',
    );
  };

  const createCredential = async (values: ShopValues) => {
    await runAction('credential-create', () => createOfflineP10Credential(values.shopId), '离线凭据 metadata 已创建');
    credentialForm.resetFields();
  };

  const startOAuth = async (values: OAuthValues) => {
    setAction('oauth-start');
    setError('');
    try {
      const result = await startOfflineP10OAuth(values.shopId, values.redirectUri);
      const state = new URL(result.authorizationUrl).searchParams.get('state') || '';
      setOfflineState(state);
      message.success('离线 OAuth state 已创建');
    } catch (nextError) {
      const nextMessage = actionError(nextError);
      setError(nextMessage);
      message.error(nextMessage);
    } finally {
      setAction('');
    }
  };

  const completeOAuth = async () => {
    if (!offlineState) return;
    await runAction('oauth-complete', () => completeOfflineP10OAuth(offlineState), '离线 OAuth 已完成');
    setOfflineState('');
  };

  const createReadRun = async (values: ShopValues) => {
    await runAction(
      'read-run',
      () => createP10InventoryReadRun(values.shopId, createP10IdempotencyKey('read-run')),
      '只读同步运行已创建',
    );
  };

  const changeGrayState = (next: 'pause' | 'stop') => {
    const revision = status?.gray?.revision;
    if (!revision) return;
    const task = next === 'pause' ? () => pauseP10Gray(revision) : () => stopP10Gray(revision);
    void runAction(`gray-${next}`, task, next === 'pause' ? 'Gray 已暂停' : 'Gray 已停止');
  };

  const rerunLastRead = () => {
    const lastRead = status?.lastRead;
    if (!lastRead) return;
    void runAction(
      'read-rerun',
      () => rerunP10InventoryRead(lastRead.runId, lastRead.revision, createP10IdempotencyKey('read-rerun')),
      '人工重跑已创建',
    );
  };

  const realReadBlocked = !status || status.currentAllowedLevel === 'L0' || !status.realInventoryReadEnabled;
  const offlineUnavailable = !status?.offlineCredentialAvailable;

  return (
    <TmPageContainer
      title="P10 人工验收控制台"
      subTitle="仓库侧开发完成，当前仅允许 L0；生产能力和真实外部激活仍被阻断。"
      extra={
        <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
          刷新
        </Button>
      }
    >
      <Spin spinning={loading}>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Alert
            showIcon
            type="warning"
            message="外部激活已阻断"
            description="未配置独立预生产基础设施和真实抖店凭据；真实 OAuth、真实读取、Gray、库存写入、Worker 与自动重试均不可用。"
          />
          {error ? <Alert showIcon type="error" message="P10 操作失败" description={error} /> : null}

          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12} lg={6}>
              <MetricCard title="允许级别" value={status?.currentAllowedLevel ?? '-'} description="仅 Fixture / Mock" intent="warning" icon={<SafetyCertificateOutlined />} />
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <MetricCard title="Provider" value={status?.realProviderEnabled ? '已启用' : '已阻断'} description={status?.providerProtocolMappingStatus ?? '-'} intent="data" icon={<ApiOutlined />} />
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <MetricCard title="有效凭据" value={credentials.filter((item) => item.status === 'active').length} description="仅 metadata 计数" intent="primary" icon={<KeyOutlined />} />
            </Col>
            <Col xs={24} sm={12} lg={6}>
              <MetricCard title="生产状态" value={status?.productionReady ? 'Ready' : 'Not Ready'} description="未通过生产验收" intent="danger" icon={<StopOutlined />} />
            </Col>
          </Row>

          <SectionCard title="运行边界" description="当前环境、只读能力、店铺范围和最后一次真实只读运行">
            <Descriptions bordered size="small" column={{ xs: 1, sm: 2, lg: 3 }}>
              <Descriptions.Item label="环境">{status?.environment ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="开发状态">{status?.developmentStatus ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="自动验证">{status?.verificationStatus ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="人工验收">{status?.manualAcceptanceStatus ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="外部激活">{status?.externalActivationStatus ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="只读代码能力">{yesNo(status?.readOnlyCapability ?? false)}</Descriptions.Item>
              <Descriptions.Item label="真实网络">{yesNo(status?.realPlatformNetworkEnabled ?? false)}</Descriptions.Item>
              <Descriptions.Item label="真实凭据">{yesNo(status?.realCredentialsEnabled ?? false)}</Descriptions.Item>
              <Descriptions.Item label="真实库存读取">{yesNo(status?.realInventoryReadEnabled ?? false)}</Descriptions.Item>
              <Descriptions.Item label="真实库存写入">{yesNo(status?.realInventoryWriteEnabled ?? false)}</Descriptions.Item>
              <Descriptions.Item label="后台 Worker">{yesNo(status?.backgroundWorkerEnabled ?? false)}</Descriptions.Item>
              <Descriptions.Item label="自动重试">{yesNo(status?.automaticRetryEnabled ?? false)}</Descriptions.Item>
              <Descriptions.Item label="租户">{status?.control.tenantId ?? '-'}</Descriptions.Item>
              <Descriptions.Item label="Allowlist 店铺">{status?.allowlist?.shopId ?? '未配置'}</Descriptions.Item>
              <Descriptions.Item label="SKU 上限">{status?.initialLimits.maxSku ?? 100}</Descriptions.Item>
              <Descriptions.Item label="最后同步">{status?.lastRead?.finishedAt ? formatDateTime(status.lastRead.finishedAt) : '暂无'}</Descriptions.Item>
              <Descriptions.Item label="最后状态">{statusTag(status?.lastRead?.status)}</Descriptions.Item>
              <Descriptions.Item label="最后错误">{status?.lastRead?.lastErrorCode || '无'}</Descriptions.Item>
              <Descriptions.Item label="限流状态">{status?.lastRead?.rateLimited ? `受限，${status.lastRead.retryAfterSeconds ?? 0}s` : '未记录限流'}</Descriptions.Item>
              <Descriptions.Item label="Request ID">{status?.lastRead?.requestId || '-'}</Descriptions.Item>
              <Descriptions.Item label="Provider Request ID">{status?.lastRead?.providerRequestId || '-'}</Descriptions.Item>
            </Descriptions>
          </SectionCard>

          <SectionCard title="Credential metadata" description="不返回或展示 Token、Ciphertext、App Secret、Client Secret">
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Alert showIcon type={offlineUnavailable ? 'info' : 'success'} message={offlineUnavailable ? '离线凭据能力未配置' : '离线凭据能力可用于人工验收'} />
              <Form<ShopValues> form={credentialForm} layout="inline" onFinish={(values) => void createCredential(values)}>
                <Form.Item name="shopId" label="店铺 ID" rules={[{ required: true, message: '请输入店铺 ID' }]}>
                  <Input placeholder="UUID" style={{ width: 300, maxWidth: '100%' }} />
                </Form.Item>
                <Form.Item>
                  <Button type="primary" htmlType="submit" disabled={!canManageCredential || offlineUnavailable} loading={action === 'credential-create'}>
                    创建离线凭据
                  </Button>
                </Form.Item>
              </Form>
              <Table<P10CredentialMetadata>
                rowKey="credentialId"
                size="small"
                pagination={false}
                scroll={{ x: 1120 }}
                dataSource={credentials}
                locale={{ emptyText: '暂无凭据 metadata' }}
                columns={[
                  { title: 'Credential ID', dataIndex: 'credentialId', width: 250, ellipsis: true },
                  { title: '店铺 ID', dataIndex: 'shopId', width: 250, ellipsis: true },
                  { title: '平台', dataIndex: 'platform', width: 100 },
                  { title: '状态', dataIndex: 'status', width: 100, render: (value) => statusTag(String(value)) },
                  { title: '版本', dataIndex: 'version', width: 80 },
                  { title: '算法', dataIndex: 'algorithm', width: 130 },
                  { title: '过期时间', dataIndex: 'expiresAt', width: 180, render: (value) => value ? formatDateTime(String(value)) : '-' },
                  {
                    title: '操作', width: 180, fixed: 'right', render: (_, row) => (
                      <Space>
                        <Popconfirm title="轮换该离线凭据？" onConfirm={() => runAction(`rotate-${row.credentialId}`, () => rotateP10Credential(row.credentialId, row.version), '凭据已轮换')}>
                          <Button size="small" disabled={!canManageCredential || offlineUnavailable || row.status === 'revoked'} loading={action === `rotate-${row.credentialId}`}>轮换</Button>
                        </Popconfirm>
                        <Popconfirm title="撤销后该凭据将立即不可用" onConfirm={() => runAction(`revoke-${row.credentialId}`, () => revokeP10Credential(row.credentialId, row.version), '凭据已撤销')}>
                          <Button danger size="small" disabled={!canManageCredential || row.status === 'revoked'} loading={action === `revoke-${row.credentialId}`}>撤销</Button>
                        </Popconfirm>
                      </Space>
                    ),
                  },
                ]}
              />
            </Space>
          </SectionCard>

          <SectionCard title="Offline OAuth" description="仅 fixture/offline 流程，不连接抖店">
            <Form<OAuthValues> form={oauthForm} layout="vertical" onFinish={(values) => void startOAuth(values)}>
              <Row gutter={[16, 0]}>
                <Col xs={24} md={10}>
                  <Form.Item name="shopId" label="店铺 ID" rules={[{ required: true, message: '请输入店铺 ID' }]}>
                    <Input placeholder="UUID" />
                  </Form.Item>
                </Col>
                <Col xs={24} md={10}>
                  <Form.Item name="redirectUri" label="回调地址" rules={[{ required: true, type: 'url', message: '请输入 allowlist 中的完整 URL' }]}>
                    <Input placeholder="http://localhost:8001/ops/p10-readiness" />
                  </Form.Item>
                </Col>
                <Col xs={24} md={4}>
                  <Form.Item label="操作">
                    <Space wrap>
                      <Button htmlType="submit" disabled={!canManageCredential || !status?.offlineOAuthEnabled} loading={action === 'oauth-start'}>创建 state</Button>
                      <Button type="primary" onClick={() => void completeOAuth()} disabled={!canManageCredential || !offlineState} loading={action === 'oauth-complete'}>完成回调</Button>
                    </Space>
                  </Form.Item>
                </Col>
              </Row>
            </Form>
          </SectionCard>

          <SectionCard title="Kill Switch" description="开关优先于 Feature Flag；Write Kill Switch 永久阻断">
            <Form<KillSwitchValues> form={killForm} layout="vertical" onFinish={confirmSwitches}>
              <Row gutter={[16, 0]}>
                {[
                  ['providerKillActive', 'Provider'],
                  ['tenantKillActive', 'Tenant'],
                  ['shopKillActive', 'Shop'],
                  ['readKillActive', 'Read'],
                ].map(([name, label]) => (
                  <Col xs={12} md={6} key={name}>
                    <Form.Item name={name} label={`${label} Kill Switch`} valuePropName="checked">
                      <Switch checkedChildren="阻断" unCheckedChildren="关闭" disabled={!canManageControl} />
                    </Form.Item>
                  </Col>
                ))}
                <Col xs={12} md={6}>
                  <Form.Item label="Write Kill Switch">{blockedTag(true)}</Form.Item>
                </Col>
              </Row>
              <Button htmlType="submit" disabled={!canManageControl} loading={action === 'switches'}>保存开关</Button>
            </Form>
          </SectionCard>

          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <SectionCard title="Tenant / Shop Allowlist" description="首版最多一个租户和一个店铺">
                <Form<ShopScopeValues> form={allowlistForm} layout="vertical" onFinish={(values) => void saveAllowlist(values)}>
                  <Form.Item name="shopId" label="店铺 ID" rules={[{ required: true, message: '请输入店铺 ID' }]}><Input placeholder="UUID" /></Form.Item>
                  <Form.Item name="enabled" label="Allowlist 状态" valuePropName="checked"><Switch checkedChildren="启用" unCheckedChildren="停用" disabled={!canManageControl} /></Form.Item>
                  <Button htmlType="submit" disabled={!canManageControl} loading={action === 'allowlist'}>保存 Allowlist</Button>
                </Form>
              </SectionCard>
            </Col>
            <Col xs={24} lg={12}>
              <SectionCard title="Gray 草案" description="仅保存范围；系统不能生成 Owner 或 Technical Lead 批准">
                <Descriptions size="small" column={1} style={{ marginBottom: 12 }}>
                  <Descriptions.Item label="状态">{statusTag(status?.gray?.status)}</Descriptions.Item>
                  <Descriptions.Item label="Owner 批准">{yesNo(status?.gray?.ownerApproved ?? false)}</Descriptions.Item>
                  <Descriptions.Item label="Technical Lead 批准">{yesNo(status?.gray?.technicalLeadApproved ?? false)}</Descriptions.Item>
                </Descriptions>
                <Form<GrayValues> form={grayForm} layout="vertical" onFinish={(values) => void saveGray(values)}>
                  <Form.Item name="shopId" label="店铺 ID" rules={[{ required: true, message: '请输入店铺 ID' }]}><Input placeholder="UUID" /></Form.Item>
                  <Form.Item name="maxSku" label="SKU 上限" rules={[{ required: true }]}><InputNumber min={1} max={100} style={{ width: '100%' }} /></Form.Item>
                  <Space wrap>
                    <Button htmlType="submit" disabled={!canManageControl} loading={action === 'gray'}>保存草案</Button>
                    <Popconfirm title="暂停 Gray 并阻断读取？" onConfirm={() => changeGrayState('pause')}>
                      <Button icon={<PauseCircleOutlined />} disabled={!canManageControl || !status?.gray} loading={action === 'gray-pause'}>暂停</Button>
                    </Popconfirm>
                    <Popconfirm title="停止 Gray 并保持阻断？" onConfirm={() => changeGrayState('stop')}>
                      <Button danger icon={<StopOutlined />} disabled={!canManageControl || !status?.gray} loading={action === 'gray-stop'}>停止</Button>
                    </Popconfirm>
                  </Space>
                </Form>
              </SectionCard>
            </Col>
          </Row>

          <SectionCard title="人工只读同步" description="仅人工触发和人工重跑；L0 下保持禁用">
            <Form<ShopValues> form={runForm} layout="inline" onFinish={(values) => void createReadRun(values)}>
              <Form.Item name="shopId" label="店铺 ID" rules={[{ required: true, message: '请输入店铺 ID' }]}>
                <Input placeholder="UUID" style={{ width: 300, maxWidth: '100%' }} />
              </Form.Item>
              <Form.Item>
                <Space wrap>
                  <Button type="primary" icon={<PlayCircleOutlined />} htmlType="submit" disabled={realReadBlocked} loading={action === 'read-run'}>发起只读同步</Button>
                  <Button
                    icon={<ReloadOutlined />}
                    disabled={realReadBlocked || status?.lastRead?.status !== 'failed'}
                    loading={action === 'read-rerun'}
                    onClick={rerunLastRead}
                  >
                    重跑最后失败运行
                  </Button>
                </Space>
              </Form.Item>
            </Form>
            {realReadBlocked ? <Typography.Text type="secondary">当前 L0 边界阻断真实 Provider 读取。</Typography.Text> : null}
          </SectionCard>
        </Space>
      </Spin>
    </TmPageContainer>
  );
}

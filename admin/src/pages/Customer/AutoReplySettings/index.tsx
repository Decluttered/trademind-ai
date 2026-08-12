import { TmPageContainer } from '@/components/ui';
import { usePermission } from '@/hooks/usePermission';
import { confirmSensitiveAction } from '@/utils/sensitiveConfirm';
import {
  getCustomerAutoReplyPolicy,
  getCustomerAutoReplySetting,
  queryCustomerAutoReplyRuns,
  updateCustomerAutoReplyPolicy,
  updateCustomerAutoReplySetting,
  type CustomerAutoReplySetting,
  type CustomerAutoReplyPolicy,
  type CustomerAutoReplyRun,
  type UpdateCustomerAutoReplyPolicy,
  type UpdateCustomerAutoReplySetting,
} from '@/services/customer';
import { queryShops } from '@/services/shops';
import { Alert, Button, Card, Form, Input, InputNumber, Select, Space, Spin, Switch, Table, Tag, Tooltip, Typography, message } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';

const TONE_OPTIONS = [
  { label: '专业', value: 'professional' },
  { label: '友好', value: 'friendly' },
  { label: '简洁', value: 'concise' },
  { label: '共情', value: 'empathetic' },
];

const STATUS_LABELS: Record<CustomerAutoReplyRun['status'], { color: string; label: string }> = {
  pending: { color: 'processing', label: '待处理' },
  generating: { color: 'processing', label: '生成中' },
  sending: { color: 'processing', label: '发送中' },
  sent: { color: 'success', label: '已发送' },
  human_required: { color: 'warning', label: '转人工' },
  skipped: { color: 'default', label: '已跳过' },
  failed: { color: 'error', label: '失败' },
};

export default function CustomerAutoReplySettingsPage() {
  const { canManageSettings, canWriteCustomer } = usePermission();
  const canManage = canManageSettings && canWriteCustomer;
  const [form] = Form.useForm<UpdateCustomerAutoReplyPolicy>();
  const [settingForm] = Form.useForm<UpdateCustomerAutoReplySetting>();
  const [shops, setShops] = useState<{ label: string; value: string }[]>([]);
  const [shopsLoading, setShopsLoading] = useState(false);
  const [shopId, setShopId] = useState<string>();
  const [policy, setPolicy] = useState<CustomerAutoReplyPolicy>();
  const [setting, setSetting] = useState<CustomerAutoReplySetting>();
  const [settingLoading, setSettingLoading] = useState(true);
  const [settingReady, setSettingReady] = useState(false);
  const [settingLoadError, setSettingLoadError] = useState<string>();
  const [runs, setRuns] = useState<CustomerAutoReplyRun[]>([]);
  const [loading, setLoading] = useState(false);
  const [policyReady, setPolicyReady] = useState(false);
  const [policyLoadError, setPolicyLoadError] = useState<string>();
  const [saving, setSaving] = useState(false);
  const [savingSetting, setSavingSetting] = useState(false);
  const policyRequestSequence = useRef(0);
  const activeShopId = useRef<string>();
  const policySubmitLocked = useRef(false);
  const settingSubmitLocked = useRef(false);
  const settingRequestSequence = useRef(0);
  const shopRequestSequence = useRef(0);
  const shopSearchTimer = useRef<ReturnType<typeof setTimeout>>();

  const loadSetting = useCallback(() => {
    const requestSequence = ++settingRequestSequence.current;
    setSettingLoading(true);
    setSettingReady(false);
    setSettingLoadError(undefined);
    settingForm.resetFields();
    void getCustomerAutoReplySetting()
      .then((row) => {
        if (requestSequence !== settingRequestSequence.current) return;
        setSetting(row);
        settingForm.setFieldsValue(row);
        setSettingReady(true);
      })
      .catch((error: unknown) => {
        if (requestSequence !== settingRequestSequence.current) return;
        setSetting(undefined);
        const errorMessage = error instanceof Error ? error.message : '运行设置加载失败';
        setSettingLoadError(errorMessage);
        message.error(errorMessage);
      })
      .finally(() => {
        if (requestSequence === settingRequestSequence.current) setSettingLoading(false);
      });
  }, [settingForm]);

  useEffect(() => {
    loadSetting();
    return () => {
      settingRequestSequence.current += 1;
    };
  }, [loadSetting]);

  const loadShops = useCallback((shopName?: string) => {
    const requestSequence = ++shopRequestSequence.current;
    setShopsLoading(true);
    void queryShops({ page: 1, pageSize: 100, shopName })
      .then((res) => {
        if (requestSequence !== shopRequestSequence.current) return;
        const rows = res.list.map((shop) => ({ label: `${shop.shopName} (${shop.platform})`, value: shop.id }));
      setShops(rows);
      setShopId((current) => current || rows[0]?.value);
      })
      .catch(() => {
        if (requestSequence === shopRequestSequence.current) setShops([]);
      })
      .finally(() => {
        if (requestSequence === shopRequestSequence.current) setShopsLoading(false);
      });
  }, []);

  useEffect(() => {
    loadShops();
    return () => {
      shopRequestSequence.current += 1;
      policyRequestSequence.current += 1;
      if (shopSearchTimer.current) clearTimeout(shopSearchTimer.current);
    };
  }, [loadShops]);

  const searchShops = (value: string) => {
    if (shopSearchTimer.current) clearTimeout(shopSearchTimer.current);
    shopSearchTimer.current = setTimeout(() => loadShops(value.trim() || undefined), 250);
  };

  useEffect(() => {
    if (!shopId) {
      activeShopId.current = undefined;
      policyRequestSequence.current += 1;
      setPolicy(undefined);
      setRuns([]);
      setPolicyReady(false);
      setPolicyLoadError(undefined);
      form.resetFields();
      return;
    }
    activeShopId.current = shopId;
    const requestSequence = ++policyRequestSequence.current;
    setPolicy(undefined);
    setRuns([]);
    setPolicyReady(false);
    setPolicyLoadError(undefined);
    form.resetFields();
    setLoading(true);
    void Promise.all([getCustomerAutoReplyPolicy(shopId), queryCustomerAutoReplyRuns(shopId)])
      .then(([row, recentRuns]) => {
        if (requestSequence !== policyRequestSequence.current || activeShopId.current !== shopId) return;
        if (row.shopId !== shopId) throw new Error('策略响应与当前店铺不匹配，请重试');
        setPolicy(row);
        setRuns(recentRuns);
        form.setFieldsValue({
          enabled: row.enabled,
          tone: row.tone,
          shopPolicy: row.shopPolicy || '',
          maxReplyRunes: row.maxReplyRunes,
          maxRepliesPerHour: row.maxRepliesPerHour,
          requireOrderContext: row.requireOrderContext,
          lowRiskOnly: true,
        });
        setPolicyReady(true);
      })
      .catch((error: unknown) => {
        if (requestSequence !== policyRequestSequence.current || activeShopId.current !== shopId) return;
        setPolicy(undefined);
        setRuns([]);
        form.resetFields();
        const errorMessage = error instanceof Error ? error.message : '自动回复策略加载失败';
        setPolicyLoadError(errorMessage);
        message.error(errorMessage);
      })
      .finally(() => {
        if (requestSequence === policyRequestSequence.current && activeShopId.current === shopId) {
          setLoading(false);
        }
      });
  }, [form, shopId]);

  const save = async (values: UpdateCustomerAutoReplyPolicy) => {
    const requestedShopId = activeShopId.current;
    if (!requestedShopId || !policyReady || policySubmitLocked.current) return;
    policySubmitLocked.current = true;
    const submit = async () => {
      setSaving(true);
      try {
        const row = await updateCustomerAutoReplyPolicy(requestedShopId, { ...values, lowRiskOnly: true });
        if (activeShopId.current !== requestedShopId || row.shopId !== requestedShopId) return;
        setPolicy(row);
        message.success(row.effectiveEnabled ? 'AI 自动回复已生效' : '策略已保存');
      } catch (error) {
        message.error(error instanceof Error ? error.message : '策略保存失败');
        throw error;
      } finally {
        policySubmitLocked.current = false;
        setSaving(false);
      }
    };
    if (!values.enabled) {
      await submit();
      return;
    }
    confirmSensitiveAction({
      title: '启用 AI 自动回复',
      content: '同步到新的低风险客户文本消息后，系统可能通过平台接口自动向买家发送 AI 回复。退款、赔付、投诉等风险内容仍转人工。',
      impacts: ['买家可见回复', 'AI 调用费用', '平台客服消息'],
      externalCall: true,
      reversible: true,
      failureHint: '可立即关闭本策略，并在失败任务中心与操作日志查看记录。',
      onOk: submit,
      onCancel: () => {
        policySubmitLocked.current = false;
      },
    });
  };

  const saveSetting = async (values: UpdateCustomerAutoReplySetting) => {
    if (!settingReady || settingSubmitLocked.current) return;
    settingSubmitLocked.current = true;
    const submit = async () => {
      setSavingSetting(true);
      try {
        const row = await updateCustomerAutoReplySetting(values);
        setSetting(row);
        settingForm.setFieldsValue(row);
        const requestedShopId = activeShopId.current;
        if (requestedShopId) {
          const currentPolicy = await getCustomerAutoReplyPolicy(requestedShopId);
          if (activeShopId.current === requestedShopId && currentPolicy.shopId === requestedShopId) {
            setPolicy(currentPolicy);
          }
        }
        message.success(row.effectiveEnabled ? '自动回复运行设置已生效' : '运行设置已保存');
      } catch (error) {
        message.error(error instanceof Error ? error.message : '运行设置保存失败');
        throw error;
      } finally {
        settingSubmitLocked.current = false;
        setSavingSetting(false);
      }
    };
    if (!values.autoReplyEnabled) {
      await submit();
      return;
    }
    confirmSensitiveAction({
      title: '开启自动回复总开关',
      content: '开启后，已启用店铺的新低风险客户消息可能由 AI 自动回复并通过平台接口发给买家。设置保存后立即生效，无需重启服务。',
      impacts: ['所有已启用店铺', '买家可见回复', 'AI 调用费用'],
      externalCall: true,
      reversible: true,
      failureHint: '可在本页立即关闭自动回复总开关，已排队任务也会在发送前重新检查。',
      onOk: submit,
      onCancel: () => {
        settingSubmitLocked.current = false;
      },
    });
  };

  return (
    <TmPageContainer
      title="AI 自动回复"
      subTitle="在页面管理消息同步、自动回复总开关和店铺策略；默认关闭，保存后动态生效。"
    >
      <Card title="运行设置" variant="borderless" style={{ marginBottom: 16 }}>
        {settingLoadError ? (
          <Alert
            showIcon
            type="error"
            message="运行设置加载失败"
            description={settingLoadError}
            action={<Button onClick={loadSetting} loading={settingLoading}>重新加载</Button>}
            style={{ marginBottom: 16 }}
          />
        ) : (
          <Alert
            showIcon
            type={setting?.effectiveEnabled ? 'warning' : 'info'}
            message={settingLoading ? '正在加载自动回复运行设置' : setting?.effectiveEnabled ? 'AI 自动回复后台任务已启用' : 'AI 自动回复后台任务未启用'}
            description={settingLoading ? '加载完成前不能修改设置。' : setting?.workerAvailable ? '设置保存在数据库中，修改后无需重启服务。' : 'Redis 或后台任务进程不可用；即使保存开启，系统也不会自动外发。'}
            style={{ marginBottom: 16 }}
          />
        )}
        <Form
          form={settingForm}
          layout="vertical"
          initialValues={{ messageSyncEnabled: false, autoReplyEnabled: false, pollIntervalSeconds: 60 }}
          disabled={!canManage || !settingReady || settingLoading || savingSetting}
          onFinish={(values) => void saveSetting(values).catch(() => undefined)}
        >
          <Space wrap size="large" align="start">
            <Form.Item name="messageSyncEnabled" label="自动同步客服消息" valuePropName="checked">
              <Switch checkedChildren="开启" unCheckedChildren="关闭" />
            </Form.Item>
            <Form.Item
              name="autoReplyEnabled"
              label="AI 自动回复总开关"
              valuePropName="checked"
              dependencies={['messageSyncEnabled']}
              rules={[({ getFieldValue }) => ({ validator: (_, value) => value && !getFieldValue('messageSyncEnabled') ? Promise.reject(new Error('请先开启自动同步客服消息')) : Promise.resolve() })]}
            >
              <Switch checkedChildren="开启" unCheckedChildren="关闭" />
            </Form.Item>
            <Form.Item name="pollIntervalSeconds" label="消息轮询间隔（秒）" rules={[{ required: true }]}>
              <InputNumber min={15} max={3600} step={15} />
            </Form.Item>
          </Space>
          <Form.Item style={{ marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={savingSetting}>保存运行设置</Button>
          </Form.Item>
        </Form>
      </Card>
      <Alert
        showIcon
        type={policy?.effectiveEnabled ? 'warning' : 'info'}
        message={policy?.effectiveEnabled ? '该店铺正在自动回复低风险消息' : '当前不会自动向买家发送消息'}
        description={
          policy?.globalEnabled
            ? policy?.workerAvailable
              ? '高风险、缺少必要订单上下文、超过频率或长度限制的消息会转人工处理。'
              : '自动回复后台任务当前不可用，系统将保持关闭；请先恢复 Redis 与后台任务进程。'
            : '页面中的 AI 自动回复总开关尚未开启；保存的店铺策略不会产生自动外发。'
        }
        style={{ marginBottom: 16 }}
      />
      {!canManage ? (
        <Alert showIcon type="warning" message="当前账号只有查看权限，不能修改自动回复策略" style={{ marginBottom: 16 }} />
      ) : null}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Typography.Text strong>店铺</Typography.Text>
          <Select
            aria-label="店铺"
            value={shopId}
            options={shops}
            showSearch
            filterOption={false}
            loading={shopsLoading}
            onSearch={searchShops}
            placeholder="选择店铺"
            style={{ minWidth: 280 }}
            onChange={setShopId}
          />
        </Space>
      </Card>
      <Spin spinning={loading}>
        {policyLoadError ? (
          <Alert showIcon type="error" message="店铺策略加载失败" description={policyLoadError} style={{ marginBottom: 16 }} />
        ) : null}
        <Card title="安全策略" variant="borderless">
          <Form
            form={form}
            layout="vertical"
            onFinish={(values) => void save(values).catch(() => undefined)}
            disabled={!shopId || !canManage || !policyReady || saving}
          >
            <Form.Item name="enabled" label="店铺自动回复" valuePropName="checked">
              <Switch checkedChildren="开启" unCheckedChildren="关闭" />
            </Form.Item>
            <Form.Item name="tone" label="回复语气" rules={[{ required: true }]}>
              <Select options={TONE_OPTIONS} />
            </Form.Item>
            <Form.Item
              name="shopPolicy"
              label="店铺政策摘要"
              extra="只填写已确认的退换货、物流、售后口径；AI 不得自行承诺退款、赔付或时效。"
            >
              <Input.TextArea rows={5} maxLength={4000} showCount />
            </Form.Item>
            <Form.Item name="requireOrderContext" label="必须关联订单后才允许自动回复" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="lowRiskOnly" label="仅低风险消息自动发送" valuePropName="checked">
              <Switch disabled />
            </Form.Item>
            <Space wrap>
              <Form.Item name="maxReplyRunes" label="单条回复最大字数" rules={[{ required: true }]}>
                <InputNumber min={50} max={2000} />
              </Form.Item>
              <Form.Item name="maxRepliesPerHour" label="每小时最多自动回复" rules={[{ required: true }]}>
                <InputNumber min={1} max={100} />
              </Form.Item>
            </Space>
            <Form.Item>
              <Button type="primary" htmlType="submit" loading={saving}>
                保存策略
              </Button>
            </Form.Item>
          </Form>
        </Card>
        <Card title="最近自动回复记录" variant="borderless" style={{ marginTop: 16 }}>
          <Table<CustomerAutoReplyRun>
            rowKey="id"
            dataSource={runs}
            pagination={false}
            scroll={{ x: 720 }}
            locale={{ emptyText: '暂无自动回复记录' }}
            columns={[
              {
                title: '状态',
                dataIndex: 'status',
                width: 110,
                render: (status: CustomerAutoReplyRun['status']) => {
                  const item = STATUS_LABELS[status] || { color: 'default', label: status };
                  return <Tag color={item.color}>{item.label}</Tag>;
                },
              },
              { title: '风险', dataIndex: 'riskLevel', width: 90, render: (value?: string) => value || '-' },
              { title: '处理原因', dataIndex: 'reasonCode', width: 190, render: (value?: string) => value || '-' },
              {
                title: '错误说明',
                dataIndex: 'errorMessage',
                width: 220,
                ellipsis: true,
                render: (value?: string) => value ? <Tooltip title={value}>{value}</Tooltip> : '-',
              },
              { title: '会话 ID', dataIndex: 'conversationId', ellipsis: true },
              { title: '创建时间', dataIndex: 'createdAt', width: 190, render: (value: string) => new Date(value).toLocaleString() },
            ]}
          />
        </Card>
      </Spin>
    </TmPageContainer>
  );
}

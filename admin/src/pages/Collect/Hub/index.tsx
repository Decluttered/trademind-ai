import {
  CloudDownloadOutlined,
  FileSearchOutlined,
  HistoryOutlined,
  LinkOutlined,
  LoginOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { history } from '@umijs/max';
import { Alert, Button, Col, List, Result, Row, Skeleton, Space, Tag, Typography, message } from 'antd';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { commonStatusLabel } from '@/constants/copywriting';
import { layoutTokens } from '@/constants/layoutTokens';
import { EmptyState, MetricCard, OperationToolbar, SectionCard, TmPageContainer } from '@/components/ui';
import { CustomCollectModal } from '@/pages/Collect/components/CustomCollectModal';
import { PinduoduoCollectModal } from '@/pages/Collect/components/PinduoduoCollectModal';
import { TaobaoTmallCollectModal } from '@/pages/Collect/components/TaobaoTmallCollectModal';
import type { CollectProviderRow, CollectProviderStatus } from '@/services/collectProviders';
import { queryCollectProviders } from '@/services/collectProviders';
import { queryCollectRules } from '@/services/collectRules';
import { fetchCollectTasks, type CollectTaskRow } from '@/services/collectTasks';
import { queryBrowserProfiles, type BrowserProfileRow } from '@/services/collectBrowserProfiles';
import { formatDateTime } from '@/utils/formatTime';
import {
  COLLECT_HUB_TYPE_HINT,
  CUSTOM_BATCH_DISABLED_TOOLTIP,
  CUSTOM_COLLECT_CARD_DESCRIPTION,
  CUSTOM_COLLECT_CARD_NOTES,
} from '@/utils/customCollectPlatform';
import {
  collectProviderStatusPresentation,
  CUSTOM_COLLECT_DISPLAY_FEATURES,
  CUSTOM_COLLECT_FEATURE_LABEL,
  NO_COLLECT_RULE_MESSAGE,
} from '@/utils/collectProviderStatus';
import {
  collectSettingsConfigButtonLabel,
  collectSettingsPath,
} from '@/utils/collectSettingsProvider';
import CollectSourceCard, {
  type CollectSourceCardCopy,
  type CollectSourceCardFeature,
} from './components/CollectSourceCard';
import { getStoredAdminLocale, translate, useLocale } from '@/locale';
import './index.less';

const { Paragraph, Text, Title } = Typography;

function featureLabelKey(feature: string): string | undefined {
  const map: Record<string, string> = {
    title: 'page.collectHub.featureTitle',
    price: 'page.collectHub.featurePrice',
    mainImages: 'page.collectHub.featureMainImages',
    descriptionImages: 'page.collectHub.featureDescriptionImages',
    attributes: 'page.collectHub.featureAttributes',
    skus: 'page.collectHub.featureSkus',
    stock: 'page.collectHub.featureStock',
  };
  return map[feature];
}

const SOURCE_ORDER = ['amazon.de', 'amazon'];

function dedicatedHubDescription(source: string): string {
  const key = source.toLowerCase();
  if (key === 'amazon.de' || key === 'amazon') {
    return translate(getStoredAdminLocale(), 'page.collectHub.amazonDescription');
  }
  return '';
}

type LoadState<T> = {
  loading: boolean;
  error?: string;
  data: T;
};

function providerRunnableForSingleTask(status: CollectProviderStatus) {
  return status === 'available' || status === 'beta';
}

function batchRowDisabledForProvider(p: CollectProviderRow): boolean {
  return !providerRunnableForSingleTask(p.status) || !p.batchSupported;
}

function batchButtonTooltipForProvider(p: CollectProviderRow): string | undefined {
  if (!providerRunnableForSingleTask(p.status)) {
    return translate(getStoredAdminLocale(), 'page.collectHub.notOpen');
  }
  if (!p.batchSupported) {
    if (p.source === 'custom') return CUSTOM_BATCH_DISABLED_TOOLTIP;
    if (p.source === 'pinduoduo' || p.source === 'pdd') {
      return 'Pinduoduo batch collection is rate-limited automatically. Start with a small test batch; some pages may require login or verification.';
    }
    if (p.source === 'taobao_tmall' || p.source === 'taobao') {
      return 'Taobao/Tmall batch collection opens each product page individually. Keep batches to 20 or fewer; complete any login or security verification before retrying.';
    }
    return p.status === 'beta' ? 'Batch collection is not available during beta' : 'This platform does not support batch collection yet';
  }
  return undefined;
}

function providerCardFeatures(p: CollectProviderRow): string[] {
  if (p.source === 'custom') {
    const fromApi = (p.features ?? []).filter((f) => f !== 'skus');
    if (fromApi.length > 0) return fromApi;
    return [...CUSTOM_COLLECT_DISPLAY_FEATURES];
  }
  if (p.source === 'pinduoduo' || p.source === 'pdd') {
    const fromApi = p.features ?? [];
    if (fromApi.length > 0) return fromApi;
    return ['title', 'price', 'mainImages', 'descriptionImages', 'attributes', 'skus'];
  }
  if (p.source === 'taobao_tmall' || p.source === 'taobao') {
    const fromApi = p.features ?? [];
    if (fromApi.length > 0) return fromApi;
    return ['title', 'price', 'mainImages', 'descriptionImages', 'attributes', 'skus'];
  }
  return p.features ?? [];
}

function featureLabelForProvider(p: CollectProviderRow, feature: string): string {
  if (p.source === 'custom') {
    return CUSTOM_COLLECT_FEATURE_LABEL[feature] ?? feature;
  }
  const key = featureLabelKey(feature);
  return key ? translate(getStoredAdminLocale(), key) : feature;
}

function providerCardCopy(p: CollectProviderRow): CollectSourceCardCopy {
  if (p.source === 'custom') {
    return {
      description: CUSTOM_COLLECT_CARD_DESCRIPTION,
      notes: CUSTOM_COLLECT_CARD_NOTES,
      typeLabel: COLLECT_HUB_TYPE_HINT.custom.title,
      typeHint: COLLECT_HUB_TYPE_HINT.custom.summary,
    };
  }
  const key = p.source.toLowerCase();
  const description = dedicatedHubDescription(key) || p.description?.trim() || '';
  const notes = p.notes?.trim() ?? '';
  return {
    description,
    notes,
    typeLabel: COLLECT_HUB_TYPE_HINT.dedicated.title,
    typeHint: '',
  };
}

function sourceOrderValue(source: string) {
  const index = SOURCE_ORDER.indexOf(source);
  return index === -1 ? SOURCE_ORDER.length : index;
}

function isLoginSensitiveSource(source: string) {
  const src = source.trim().toLowerCase();
  return src === 'pinduoduo' || src === 'pdd' || src === 'taobao_tmall' || src === 'taobao' || src === 'custom';
}

function taskStatusTagColor(status: string) {
  const key = status.trim().toLowerCase();
  if (key === 'success') return 'success';
  if (key === 'failed') return 'error';
  if (key === 'running' || key === 'retrying') return 'processing';
  if (key === 'pending') return 'default';
  return 'default';
}

async function openCustomCollectModal(
  setCustomModalOpen: (open: boolean) => void,
): Promise<void> {
  try {
    const res = await queryCollectRules({ page: 1, pageSize: 1, status: 'enabled' });
    if (!res.list?.length) {
      message.warning(NO_COLLECT_RULE_MESSAGE);
    }
  } catch {
    // Still open the Modal; the dialog guides the user to create a rule
  }
  setCustomModalOpen(true);
}

function RecentTaskList({
  loading,
  error,
  tasks,
}: {
  loading: boolean;
  error?: string;
  tasks: CollectTaskRow[];
}) {
  if (loading) {
    return <Skeleton active paragraph={{ rows: 4 }} />;
  }
  if (error) {
    return (
      <Result
        status="warning"
        title="Recent tasks could not be loaded"
        subTitle="Open the collection task list to check task status, or reopen Collect Hub later."
        extra={
          <Button onClick={() => history.push('/collect/tasks')}>
            View collection tasks
          </Button>
        }
      />
    );
  }
  if (!tasks.length) {
    return (
      <EmptyState
        compact
        title="No recent collection tasks"
        description="Choose a source and submit a product link above. Tasks will appear here."
        actionLabel="Start collecting products"
        actionPath="/collect/hub"
      />
    );
  }

  return (
    <List
      className="tm-collect-hub-task-list"
      dataSource={tasks}
      renderItem={(task) => (
        <List.Item
          actions={[
            <Button key="view" type="link" onClick={() => history.push(`/collect/tasks?source=${encodeURIComponent(task.source)}`)}>
              View
            </Button>,
          ]}
        >
          <List.Item.Meta
            title={
              <Space wrap size={8} className="tm-collect-hub-task-list__title">
                <Text strong>{task.source}</Text>
                <Tag color={taskStatusTagColor(task.status)}>{commonStatusLabel(task.status)}</Tag>
              </Space>
            }
            description={
              <div className="tm-collect-hub-task-list__description">
                <Text
                  type="secondary"
                  ellipsis={{ tooltip: task.sourceUrl }}
                  className="tm-collect-hub-task-list__url"
                >
                  {task.sourceUrl}
                </Text>
                <Text type="secondary">{formatDateTime(task.createdAt)}</Text>
              </div>
            }
          />
        </List.Item>
      )}
    />
  );
}

function BrowserProfileSummary({
  loading,
  error,
  profiles,
}: {
  loading: boolean;
  error?: string;
  profiles: BrowserProfileRow[];
}) {
  if (loading) {
    return <Skeleton active paragraph={{ rows: 3 }} />;
  }
  if (error) {
    return (
      <Alert
        type="warning"
        showIcon
        message="Login state could not be loaded"
        description="For platforms that require login or verification, recheck the collector browser login state before collecting."
        action={<Button size="small" onClick={() => history.push('/collect/browser-profiles')}>Open login state</Button>}
      />
    );
  }
  if (!profiles.length) {
    return (
      <EmptyState
        compact
        title="No saved login state"
        description="Taobao/Tmall, Pinduoduo, and some custom sites may require login before collection."
        actionLabel="Manage login state"
        actionPath="/collect/browser-profiles"
      />
    );
  }
  return (
    <Space direction="vertical" size={10} className="tm-collect-hub-profile-list">
      {profiles.map((profile) => (
        <div className="tm-collect-hub-profile" key={profile.id}>
          <div className="tm-collect-hub-profile__main">
            <Text strong className="tm-collect-hub-profile__name">
              {profile.name}
            </Text>
            <Text type="secondary" className="tm-collect-hub-profile__domain">
              {profile.domain}
            </Text>
          </div>
          <Tag color={profile.lastCheckStatus === 'public' ? 'success' : 'warning'}>
            {profile.lastCheckStatus || profile.status}
          </Tag>
        </div>
      ))}
    </Space>
  );
}

export default function CollectHubPage() {
  const { t } = useLocale();
  const [providerState, setProviderState] = useState<LoadState<CollectProviderRow[]>>({
    loading: true,
    data: [],
  });
  const [recentState, setRecentState] = useState<LoadState<CollectTaskRow[]>>({
    loading: true,
    data: [],
  });
  const [failedTotal, setFailedTotal] = useState<number | undefined>();
  const [profileState, setProfileState] = useState<LoadState<BrowserProfileRow[]>>({
    loading: true,
    data: [],
  });
  const [customModalOpen, setCustomModalOpen] = useState(false);
  const [pddModalOpen, setPddModalOpen] = useState(false);
  const [tbModalOpen, setTbModalOpen] = useState(false);

  const loadProviders = useCallback(async (isActive: () => boolean = () => true) => {
    setProviderState((state) => ({ ...state, loading: true, error: undefined }));
    try {
      const rows = await queryCollectProviders();
      if (isActive()) {
        setProviderState({ loading: false, data: Array.isArray(rows) ? rows : [] });
      }
    } catch (error) {
      if (isActive()) {
        setProviderState({
          loading: false,
          data: [],
          error: error instanceof Error ? error.message : 'Collection sources could not be loaded.',
        });
      }
    }
  }, []);

  useEffect(() => {
    let active = true;
    void loadProviders(() => active);
    return () => {
      active = false;
    };
  }, [loadProviders]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setRecentState((state) => ({ ...state, loading: true, error: undefined }));
      try {
        const [recent, failed] = await Promise.all([
          fetchCollectTasks({ page: 1, pageSize: 5 }),
          fetchCollectTasks({ page: 1, pageSize: 1, status: 'failed' }),
        ]);
        if (!cancelled) {
          setRecentState({ loading: false, data: recent.list ?? [] });
          setFailedTotal(failed.pagination?.total ?? 0);
        }
      } catch (error) {
        if (!cancelled) {
          setRecentState({
            loading: false,
            data: [],
            error: error instanceof Error ? error.message : 'Recent collection tasks could not be loaded.',
          });
          setFailedTotal(undefined);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setProfileState((state) => ({ ...state, loading: true, error: undefined }));
      try {
        const res = await queryBrowserProfiles({ page: 1, pageSize: 4, status: 'active' });
        if (!cancelled) {
          setProfileState({ loading: false, data: res.list ?? [] });
        }
      } catch (error) {
        if (!cancelled) {
          setProfileState({
            loading: false,
            data: [],
            error: error instanceof Error ? error.message : 'Login state could not be loaded.',
          });
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const sortedProviders = useMemo(
    () =>
      [...providerState.data].sort(
        (a, b) => sourceOrderValue(a.source) - sourceOrderValue(b.source) || a.name.localeCompare(b.name),
      ),
    [providerState.data],
  );

  const runnableProviders = sortedProviders.filter((provider) => providerRunnableForSingleTask(provider.status));
  const batchProviders = sortedProviders.filter((provider) => providerRunnableForSingleTask(provider.status) && provider.batchSupported);
  const loginSensitiveProviders = sortedProviders.filter((provider) => isLoginSensitiveSource(provider.source));
  const primaryProvider = runnableProviders[0];

  const openSingleCollect = (provider: CollectProviderRow) => {
    if (provider.source === 'custom') {
      void openCustomCollectModal(setCustomModalOpen);
    } else if (provider.source === 'pinduoduo' || provider.source === 'pdd') {
      setPddModalOpen(true);
    } else if (provider.source === 'taobao_tmall' || provider.source === 'taobao') {
      setTbModalOpen(true);
    } else {
      history.push(`/collect/tasks?source=${encodeURIComponent(provider.source)}`);
    }
  };

  const pageExtra = (
    <OperationToolbar>
      <Button icon={<HistoryOutlined />} onClick={() => history.push('/collect/tasks')}>
        Collection tasks
      </Button>
      <Button icon={<SettingOutlined />} onClick={() => history.push('/settings/collector')}>
        Collector settings
      </Button>
    </OperationToolbar>
  );

  return (
    <TmPageContainer
      title={t('page.collectHub.title')}
      subTitle={t('page.collectHub.description')}
      contentMaxWidth={layoutTokens.dashboardMaxWidth}
      extra={pageExtra}
    >
      <div className="tm-collect-hub">
        <section className="tm-collect-hub-hero">
          <div className="tm-collect-hub-hero__main">
            <Text className="tm-collect-hub-hero__eyebrow">Cross-border product collection</Text>
            <Title level={4} className="tm-collect-hub-hero__title">
              Choose a source, then turn product links into operational drafts.
            </Title>
            <Paragraph className="tm-collect-hub-hero__desc">
              Collection tasks enter a queue. If login, verification, or platform limits interrupt work, check collector browser login state, then retry or recover failed tasks in bulk.
            </Paragraph>
            <OperationToolbar>
              <Button
                type="primary"
                size="large"
                icon={<CloudDownloadOutlined />}
                disabled={!primaryProvider || providerState.loading}
                onClick={() => primaryProvider && openSingleCollect(primaryProvider)}
              >
                Start collecting products
              </Button>
              <Button size="large" icon={<FileSearchOutlined />} onClick={() => history.push('/collect/batches')}>
                Batch collection
              </Button>
              <Button size="large" type="link" onClick={() => history.push('/collect/rules')}>
                Manage collection rules
              </Button>
            </OperationToolbar>
          </div>
          <div className="tm-collect-hub-hero__side">
            <MetricCard
              title="Supported sources"
              value={providerState.loading ? '—' : sortedProviders.length}
              description="Based on collectors returned by the API"
              intent="primary"
              icon={<LinkOutlined />}
            />
            <MetricCard
              title="Single-item collection"
              value={providerState.loading ? '—' : runnableProviders.length}
              description="Available or in beta"
              intent="success"
              icon={<CloudDownloadOutlined />}
            />
            <MetricCard
              title="Batch entry points"
              value={providerState.loading ? '—' : batchProviders.length}
              description="Sources that accept batch submission"
              intent="data"
              icon={<FileSearchOutlined />}
            />
            <MetricCard
              title="Failed tasks to recover"
              value={recentState.loading ? '—' : failedTotal ?? '—'}
              description="From the collection tasks API"
              intent="warning"
              icon={<WarningOutlined />}
              onClick={() => history.push('/collect/tasks?status=failed')}
            />
          </div>
        </section>

        <Row gutter={[16, 16]}>
          <Col xs={24} lg={16}>
            <SectionCard
              title="Collection sources"
              description="Use dedicated collectors for supported platforms first. Custom collectors are for unsupported sites; test their rules before use."
              headerExtra={
                <Button
                  icon={<ReloadOutlined />}
                  loading={providerState.loading}
                  onClick={() => void loadProviders()}
                >
                  Reload
                </Button>
              }
            >
              {providerState.loading ? (
                <Row gutter={[16, 16]}>
                  {Array.from({ length: 3 }).map((_, index) => (
                    <Col xs={24} md={12} xl={8} key={index}>
                      <Skeleton active paragraph={{ rows: 6 }} />
                    </Col>
                  ))}
                </Row>
              ) : providerState.error ? (
                <Result
                  status="warning"
                  title="Collection sources could not be loaded"
                  subTitle="Check collector service configuration or try again later. Existing tasks remain available in the collection task list."
                  extra={
                    <Space wrap>
                      <Button type="primary" onClick={() => history.push('/settings/collector')}>
                        Check collector settings
                      </Button>
                      <Button onClick={() => history.push('/collect/tasks')}>
                        View collection tasks
                      </Button>
                    </Space>
                  }
                />
              ) : sortedProviders.length === 0 ? (
                <EmptyState
                  title="暂无可用采集来源"
                  description="请先到采集设置中检查采集服务配置，或确认后端已返回采集器列表。"
                  actionLabel="检查采集设置"
                  actionPath="/settings/collector"
                />
              ) : (
                <Row gutter={[16, 16]}>
                  {sortedProviders.map((provider) => {
                    const statusTag = collectProviderStatusPresentation(provider.source, provider.status);
                    const copy = providerCardCopy(provider);
                    const features: CollectSourceCardFeature[] = providerCardFeatures(provider).map((feature) => ({
                      key: feature,
                      label: featureLabelForProvider(provider, feature),
                    }));
                    return (
                      <Col xs={24} md={12} xl={8} key={provider.source}>
                        <CollectSourceCard
                          provider={provider}
                          copy={copy}
                          statusTag={statusTag}
                          features={features}
                          singleDisabled={!providerRunnableForSingleTask(provider.status)}
                          singleTooltip={providerRunnableForSingleTask(provider.status) ? undefined : '当前版本暂未开放'}
                          batchDisabled={batchRowDisabledForProvider(provider)}
                          batchTooltip={batchButtonTooltipForProvider(provider)}
                          onSingleCollect={() => openSingleCollect(provider)}
                          onBatchCollect={() => history.push(`/collect/batches?source=${encodeURIComponent(provider.source)}`)}
                          onSettings={() => history.push(collectSettingsPath(provider.source))}
                          settingsLabel={collectSettingsConfigButtonLabel(provider.status)}
                        />
                      </Col>
                    );
                  })}
                </Row>
              )}
            </SectionCard>
          </Col>

          <Col xs={24} lg={8}>
            <Space direction="vertical" size={16} className="tm-collect-hub__side-stack">
              <SectionCard
                title="登录与验证风险"
                description="平台登录态和验证码会影响采集成功率。"
                compact
              >
                <div className="tm-collect-hub-risk-list">
                  {loginSensitiveProviders.map((provider) => (
                    <div className="tm-collect-hub-risk" key={provider.source}>
                      <LoginOutlined />
                      <div className="tm-collect-hub-risk__content">
                        <Text strong>{provider.name}</Text>
                        <Text type="secondary">可能需要采集浏览器登录或人工验证</Text>
                      </div>
                    </div>
                  ))}
                </div>
              </SectionCard>

              <SectionCard
                title="浏览器登录状态"
                description="用于处理登录页、验证码和安全验证。"
                headerExtra={
                  <Button type="link" onClick={() => history.push('/collect/browser-profiles')}>
                    管理
                  </Button>
                }
                compact
              >
                <BrowserProfileSummary
                  loading={profileState.loading}
                  error={profileState.error}
                  profiles={profileState.data}
                />
              </SectionCard>

              <SectionCard
                title="失败恢复"
                description="失败任务保留原因和重试入口。"
                compact
              >
                <div className="tm-collect-hub-recovery">
                  <SafetyCertificateOutlined />
                  <div>
                    <Text strong>{failedTotal ?? '—'} 个失败任务</Text>
                    <Text type="secondary">进入任务列表查看原因、登录提示和重试入口。</Text>
                  </div>
                  <Button onClick={() => history.push('/collect/tasks?status=failed')}>去处理</Button>
                </div>
              </SectionCard>
            </Space>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} lg={16}>
            <SectionCard
              title="最近采集任务"
              description="用于确认任务是否已进入队列，以及失败后下一步处理方式。"
              headerExtra={
                <Button type="link" onClick={() => history.push('/collect/tasks')}>
                  查看全部
                </Button>
              }
            >
              <RecentTaskList
                loading={recentState.loading}
                error={recentState.error}
                tasks={recentState.data}
              />
            </SectionCard>
          </Col>
          <Col xs={24} lg={8}>
            <SectionCard title="相关管理入口" description="采集前后的常用配置和记录。" compact>
              <div className="tm-collect-hub-links">
                <Button block onClick={() => history.push('/collect/tasks')}>采集任务</Button>
                <Button block onClick={() => history.push('/collect/batches')}>批量采集</Button>
                <Button block onClick={() => history.push('/collect/browser-profiles')}>浏览器登录状态</Button>
                <Button block onClick={() => history.push('/collect/rules')}>采集规则</Button>
                <Button block onClick={() => history.push('/collect/monitor')}>采集监控</Button>
                <Button block onClick={() => history.push('/settings/collector')}>采集设置</Button>
              </div>
            </SectionCard>
          </Col>
        </Row>
      </div>

      <CustomCollectModal open={customModalOpen} onClose={() => setCustomModalOpen(false)} />
      <PinduoduoCollectModal open={pddModalOpen} onClose={() => setPddModalOpen(false)} />
      <TaobaoTmallCollectModal open={tbModalOpen} onClose={() => setTbModalOpen(false)} />
    </TmPageContainer>
  );
}

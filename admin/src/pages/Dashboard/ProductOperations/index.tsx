import {
  ArrowRightOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  CloudUploadOutlined,
  DatabaseOutlined,
  FileTextOutlined,
  NotificationOutlined,
  PictureOutlined,
  ReloadOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  ShopOutlined,
  UnorderedListOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import { MetricCard, OperationToolbar, TmPageContainer, type MetricCardIntent } from '@/components/ui';
import { useListEmptyLocale } from '@/hooks/useListEmptyLocale';
import { useLocale } from '@/locale';
import { formatDateTime } from '@/utils/formatTime';
import { history } from '@umijs/max';
import {
  Button,
  Col,
  DatePicker,
  Result,
  Row,
  Select,
  Skeleton,
  Space,
  Tag,
  Typography,
} from 'antd';
import dayjs, { type Dayjs } from 'dayjs';
import type { ReactNode } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { layoutTokens } from '@/constants/layoutTokens';
import { PLATFORM_OPTIONS } from '@/constants/userFriendly';
import {
  DEFAULT_FUNNEL,
  DEFAULT_QUICK_LINKS,
  EMPTY_SUMMARY,
  mergeExceptions,
  mergeFunnel,
  mergeTodos,
} from '@/constants/dashboardDefaults';
import {
  formatRecentItem,
  recentStatusColor,
  recentStatusLabel,
  recentTranslateWarningSubtitle,
} from '@/constants/dashboardRecent';
import {
  queryProductOperationDashboard,
  type DashboardException,
  type DashboardFunnelStep,
  type DashboardRecentItem,
  type DashboardSummary,
  type DashboardTodo,
  type ProductOperationDashboard,
} from '@/services/dashboard';
import { queryShops, type ShopListRow } from '@/services/shops';
import { useUrlQueryState } from '@/hooks/useUrlState';
import { appendSourceToUrl, resolveProductSourceFromQuery } from '@/utils/urlState';

const { RangePicker } = DatePicker;

type Translate = ReturnType<typeof useLocale>['t'];

function sourceOptions(t: Translate) {
  return [
    { label: 'Amazon.de', value: 'amazon.de' },
    { label: t('dashboard.sources.pinduoduo'), value: 'pinduoduo' },
    { label: t('dashboard.sources.custom'), value: 'custom' },
    { label: t('dashboard.sources.aliexpress'), value: 'aliexpress' },
    { label: t('dashboard.sources.manual'), value: 'manual' },
  ];
}

const RECENT_TYPE_META: Record<string, { icon: ReactNode; color: string; bg: string }> = {
  collect: { icon: <CloudUploadOutlined />, color: 'var(--ant-color-primary)', bg: 'var(--ant-color-primary-bg)' },
  ai_task: { icon: <RobotOutlined />, color: 'var(--tm-ai-accent)', bg: 'var(--ant-color-primary-bg)' },
  ai_batch: { icon: <FileTextOutlined />, color: 'var(--tm-ai-accent)', bg: 'var(--ant-color-primary-bg)' },
  image_task: { icon: <PictureOutlined />, color: 'var(--ant-color-info)', bg: 'var(--ant-color-info-bg)' },
  product_publish: { icon: <ShopOutlined />, color: 'var(--ant-color-success)', bg: 'var(--ant-color-success-bg)' },
  inventory_alert: { icon: <WarningOutlined />, color: 'var(--ant-color-warning)', bg: 'var(--ant-color-warning-bg)' },
  failed_publish: { icon: <ShopOutlined />, color: 'var(--ant-color-error)', bg: 'var(--ant-color-error-bg)' },
  failed_inventory_sync: { icon: <WarningOutlined />, color: 'var(--ant-color-error)', bg: 'var(--ant-color-error-bg)' },
  failed_collect: { icon: <CloudUploadOutlined />, color: 'var(--ant-color-error)', bg: 'var(--ant-color-error-bg)' },
  task_alert: { icon: <NotificationOutlined />, color: 'var(--ant-color-warning)', bg: 'var(--ant-color-warning-bg)' },
  failed: { icon: <WarningOutlined />, color: 'var(--ant-color-error)', bg: 'var(--ant-color-error-bg)' },
};

const ellipsizedText: React.CSSProperties = {
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
  display: 'block',
  maxWidth: '100%',
};

function RecentActivityRow({
  item,
  bucket,
}: {
  item: DashboardRecentItem;
  bucket: string;
}) {
  const { t } = useLocale();
  const meta = RECENT_TYPE_META[item.type] ?? RECENT_TYPE_META.image_task;
  const { title, subtitle } = formatRecentItem(item);
  const statusLabel = recentStatusLabel(item.status);
  const statusColor = recentStatusColor(item.status);
  const displaySubtitle =
    item.type === 'image_task' ? recentTranslateWarningSubtitle(subtitle) ?? subtitle : subtitle;

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => history.push(appendSourceToUrl(item.link))}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') history.push(appendSourceToUrl(item.link));
      }}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 14,
        padding: '14px 16px',
        borderRadius: 10,
        border: '1px solid var(--ant-color-border-secondary)',
        background: 'var(--ant-color-bg-container)',
        cursor: 'pointer',
        transition: 'border-color 0.2s, box-shadow 0.2s',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = meta.color;
        e.currentTarget.style.boxShadow = 'var(--tm-shadow-card)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = 'var(--ant-color-border-secondary)';
        e.currentTarget.style.boxShadow = 'none';
      }}
    >
      <div
        style={{
          width: 40,
          height: 40,
          borderRadius: 12,
          background: meta.bg,
          color: meta.color,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 18,
          flexShrink: 0,
        }}
      >
        {meta.icon}
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <Space wrap size={6} style={{ marginBottom: 6 }}>
          <Tag
            bordered={false}
            style={{ margin: 0, background: meta.bg, color: meta.color, fontSize: 12 }}
          >
            {bucket}
          </Tag>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            {formatDateTime(item.occurredAt)}
          </Typography.Text>
          {item.status ? <Tag color={statusColor}>{statusLabel}</Tag> : null}
        </Space>
        <Typography.Text strong style={{ ...ellipsizedText, fontSize: 14 }} title={title}>
          {title}
        </Typography.Text>
        {displaySubtitle ? (
          <Typography.Text
            type="secondary"
            style={{ ...ellipsizedText, fontSize: 12, marginTop: 4 }}
            title={displaySubtitle}
          >
            {displaySubtitle}
          </Typography.Text>
        ) : null}
      </div>
      <Button
        type="link"
        size="small"
        icon={<ArrowRightOutlined />}
        onClick={(e) => {
          e.stopPropagation();
          history.push(appendSourceToUrl(item.link));
        }}
      >
        {t('dashboard.actions.view')}
      </Button>
    </div>
  );
}

const RECENT_TYPE_KEYS: Record<string, string> = {
  collect: 'dashboard.recent.collect',
  ai_task: 'dashboard.recent.aiTask',
  ai_batch: 'dashboard.recent.aiBatch',
  image_task: 'dashboard.recent.imageTask',
  product_publish: 'dashboard.recent.productPublish',
  inventory_alert: 'dashboard.recent.inventoryAlert',
  failed_publish: 'dashboard.recent.failedPublish',
  failed_inventory_sync: 'dashboard.recent.failedInventorySync',
  failed_collect: 'dashboard.recent.failedCollect',
  task_alert: 'dashboard.recent.taskAlert',
};

const TODO_ACTION_KEYS: Record<string, string> = {
  missing_ai_title: 'dashboard.todoActions.optimize',
  missing_ai_description: 'dashboard.todoActions.generate',
  readiness_blocked: 'dashboard.todoActions.review',
  publishable: 'dashboard.todoActions.publish',
  inventory_alerts: 'dashboard.todoActions.handle',
  ai_image_failed: 'dashboard.todoActions.view',
  collect_failed: 'dashboard.todoActions.retry',
  publish_failed: 'dashboard.todoActions.handle',
  order_exceptions: 'dashboard.todoActions.handle',
  failures: 'dashboard.todoActions.view',
};

type FilterState = {
  range?: [Dayjs, Dayjs];
  platform?: string;
  shopId?: string;
  source?: string;
};

const DASHBOARD_QUERY_KEYS = ['start', 'end', 'platform', 'shopId', 'productSource', 'source'] as const;

function buildDashboardFiltersFromUrl(
  urlState: Record<(typeof DASHBOARD_QUERY_KEYS)[number], string | undefined>,
): FilterState {
  return {
    range: parseRange(urlState.start, urlState.end),
    platform: urlState.platform,
    shopId: urlState.shopId,
    source: resolveProductSourceFromQuery(urlState.productSource, urlState.source),
  };
}

function dashboardFiltersToUrlPatch(filters: FilterState) {
  const [start, end] = filters.range ?? [];
  return {
    start: start ? start.startOf('day').toISOString() : undefined,
    end: end ? end.endOf('day').toISOString() : undefined,
    platform: filters.platform,
    shopId: filters.shopId,
    productSource: filters.source,
  };
}

function sameDashboardUrlPatch(
  a: ReturnType<typeof dashboardFiltersToUrlPatch>,
  urlState: Record<(typeof DASHBOARD_QUERY_KEYS)[number], string | undefined>,
) {
  return (
    (a.start || undefined) === (urlState.start || undefined) &&
    (a.end || undefined) === (urlState.end || undefined) &&
    (a.platform || undefined) === (urlState.platform || undefined) &&
    (a.shopId || undefined) === (urlState.shopId || undefined) &&
    (a.productSource || undefined) === (urlState.productSource || undefined)
  );
}

function parseRange(start?: string, end?: string): [Dayjs, Dayjs] | undefined {
  if (!start || !end) return undefined;
  const s = dayjs(start);
  const e = dayjs(end);
  if (!s.isValid() || !e.isValid()) return undefined;
  return [s, e];
}

function TodoCardItem({ item }: { item: DashboardTodo }) {
  const { t } = useLocale();
  const actionLabel = t(TODO_ACTION_KEYS[item.key] ?? 'dashboard.todoActions.handle');
  const hasCount = (item.count ?? 0) > 0;
  return (
    <ProCard
      variant="outlined"
      bodyStyle={{ padding: '16px', height: '100%' }}
      style={hasCount ? { borderColor: 'var(--ant-color-warning)' } : undefined}
    >
      <Space direction="vertical" size={8} style={{ width: '100%' }}>
        <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
          <Typography.Text strong>{item.title}</Typography.Text>
          <Typography.Title level={3} style={{ margin: 0 }}>
            {item.count ?? 0}
          </Typography.Title>
        </Space>
        <Button type="primary" block onClick={() => history.push(appendSourceToUrl(item.link))}>
          {actionLabel}
        </Button>
      </Space>
    </ProCard>
  );
}

function ExceptionRow({ item }: { item: DashboardException }) {
  const { t } = useLocale();
  return (
    <ProCard
      variant="outlined"
      bodyStyle={{ padding: '16px 18px' }}
      style={{ margin: 0 }}
      hoverable
      onClick={() => history.push(appendSourceToUrl(item.link))}
    >
      <Row align="middle" gutter={16} wrap={false}>
        <Col flex="auto">
          <Space direction="vertical" size={6}>
            <Space>
              <Typography.Text strong>{item.title}</Typography.Text>
              <Tag color={item.count > 0 ? 'error' : 'default'}>{item.count ?? 0}</Tag>
            </Space>
            {item.lastOccurredAt ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                {t('dashboard.recently')}: {formatDateTime(item.lastOccurredAt)}
              </Typography.Text>
            ) : null}
          </Space>
        </Col>
        <Col>
          <Button type="link" icon={<ArrowRightOutlined />}>
            {t('dashboard.todoActions.handle')}
          </Button>
        </Col>
      </Row>
    </ProCard>
  );
}

const QUICK_LINK_META: Record<string, { icon: ReactNode; color: string; bg: string }> = {
  '/collect/hub': { icon: <CloudUploadOutlined />, color: 'var(--ant-color-primary)', bg: 'var(--ant-color-primary-bg)' },
  '/product/drafts': { icon: <FileTextOutlined />, color: 'var(--ant-color-primary)', bg: 'var(--ant-color-primary-bg)' },
  '/ai/text-batches': { icon: <RobotOutlined />, color: 'var(--tm-ai-accent)', bg: 'var(--ant-color-primary-bg)' },
  '/ai/image-tasks': { icon: <PictureOutlined />, color: 'var(--ant-color-info)', bg: 'var(--ant-color-info-bg)' },
  '/product/drafts?readiness=blocked': { icon: <SafetyCertificateOutlined />, color: 'var(--ant-color-warning)', bg: 'var(--ant-color-warning-bg)' },
  '/product/publish-tasks': { icon: <ShopOutlined />, color: 'var(--ant-color-success)', bg: 'var(--ant-color-success-bg)' },
  '/inventory/alerts': { icon: <WarningOutlined />, color: 'var(--ant-color-error)', bg: 'var(--ant-color-error-bg)' },
  '/ops/task-center/failures': { icon: <CloseCircleOutlined />, color: 'var(--ant-color-error)', bg: 'var(--ant-color-error-bg)' },
  '/orders/exceptions': { icon: <UnorderedListOutlined />, color: 'var(--ant-color-warning)', bg: 'var(--ant-color-warning-bg)' },
  '/settings/ai': { icon: <SettingOutlined />, color: 'var(--tm-ai-accent)', bg: 'var(--ant-color-primary-bg)' },
  '/settings/image': { icon: <PictureOutlined />, color: 'var(--ant-color-info)', bg: 'var(--ant-color-info-bg)' },
  '/settings/storage': { icon: <DatabaseOutlined />, color: 'var(--ant-color-text-secondary)', bg: 'var(--ant-color-fill-quaternary)' },
};

const QUICK_LINK_GROUPS: { labelKey: string; links: string[] }[] = [
  {
    labelKey: 'dashboard.quickLinkGroups.productOperations',
    links: [
      '/collect/hub',
      '/product/drafts',
      '/product/drafts?readiness=blocked',
      '/product/publish-tasks',
      '/inventory/alerts',
    ],
  },
  {
    labelKey: 'dashboard.quickLinkGroups.aiTools',
    links: ['/ai/text-batches', '/ai/image-tasks'],
  },
  {
    labelKey: 'dashboard.quickLinkGroups.operationsAndSettings',
    links: [
      '/ops/task-center/failures',
      '/orders/exceptions',
      '/settings/ai',
      '/settings/image',
      '/settings/storage',
    ],
  },
];

const DASHBOARD_ITEM_TITLE_KEYS: Record<string, string> = {
  missing_ai_title: 'dashboard.items.missingAiTitle',
  missing_ai_description: 'dashboard.items.missingAiDescription',
  readiness_blocked: 'dashboard.items.readinessBlocked',
  publishable: 'dashboard.items.publishable',
  inventory_alerts: 'dashboard.items.inventoryAlerts',
  ai_image_failed: 'dashboard.items.aiImageFailed',
  collect_failed: 'dashboard.items.collectFailed',
  publish_failed: 'dashboard.items.publishFailed',
  customer_pending: 'dashboard.items.customerPending',
  failed_tasks: 'dashboard.items.failedTasks',
  config_incomplete: 'dashboard.items.configIncomplete',
  order_exceptions: 'dashboard.items.orderExceptions',
  collected: 'dashboard.items.collected',
  draft: 'dashboard.items.drafts',
  ai_text: 'dashboard.items.aiText',
  ai_image: 'dashboard.items.aiImage',
  readiness_pass: 'dashboard.items.readinessPassed',
  published: 'dashboard.items.published',
  ai_text_failed: 'dashboard.items.aiTextFailed',
  inventory_sync_failed: 'dashboard.items.inventorySyncFailed',
};

const QUICK_LINK_TITLE_KEYS: Record<string, string> = {
  '/collect/hub': 'dashboard.quickLinks.collectHub',
  '/product/drafts': 'dashboard.quickLinks.drafts',
  '/ai/text-batches': 'dashboard.quickLinks.aiTextBatches',
  '/ai/image-tasks': 'dashboard.quickLinks.aiImageTasks',
  '/product/drafts?readiness=blocked': 'dashboard.quickLinks.readinessChecks',
  '/product/publish-tasks': 'dashboard.quickLinks.publishTasks',
  '/inventory/alerts': 'dashboard.quickLinks.inventoryAlerts',
  '/ops/task-center/failures': 'dashboard.quickLinks.failedTasks',
  '/orders/exceptions': 'dashboard.quickLinks.orderExceptions',
  '/customer/hub': 'dashboard.quickLinks.customerHub',
  '/settings/config-status': 'dashboard.quickLinks.configStatus',
  '/settings/ai': 'dashboard.quickLinks.aiSettings',
  '/settings/image': 'dashboard.quickLinks.imageSettings',
  '/settings/storage': 'dashboard.quickLinks.storageSettings',
};

function localizeDashboardItems<T extends { key: string; title: string }>(items: T[], t: Translate): T[] {
  return items.map((item) => {
    const key = DASHBOARD_ITEM_TITLE_KEYS[item.key];
    return key ? { ...item, title: t(key) } : item;
  });
}

function localizeQuickLinks<T extends { title: string; link: string }>(items: T[], t: Translate): T[] {
  return items.map((item) => {
    const key = QUICK_LINK_TITLE_KEYS[item.link];
    return key ? { ...item, title: t(key) } : item;
  });
}

function QuickLinkCard(props: { title: string; link: string }) {
  const meta = QUICK_LINK_META[props.link] ?? {
    icon: <ArrowRightOutlined />,
    color: 'var(--ant-color-text-secondary)',
    bg: 'var(--ant-color-fill-quaternary)',
  };

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => history.push(appendSourceToUrl(props.link))}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') history.push(appendSourceToUrl(props.link));
      }}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        height: '100%',
        minHeight: 56,
        padding: '12px 14px',
        borderRadius: 10,
        border: '1px solid var(--ant-color-border-secondary)',
        background: 'var(--ant-color-bg-container)',
        cursor: 'pointer',
        transition: 'border-color 0.2s, box-shadow 0.2s, transform 0.15s',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.borderColor = meta.color;
        e.currentTarget.style.boxShadow = 'var(--tm-shadow-elevated)';
        e.currentTarget.style.transform = 'translateY(-1px)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.borderColor = 'var(--ant-color-border-secondary)';
        e.currentTarget.style.boxShadow = 'none';
        e.currentTarget.style.transform = 'none';
      }}
    >
      <div
        style={{
          width: 36,
          height: 36,
          borderRadius: 10,
          background: meta.bg,
          color: meta.color,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 17,
          flexShrink: 0,
        }}
      >
        {meta.icon}
      </div>
      <Typography.Text strong style={{ flex: 1, fontSize: 13, lineHeight: 1.4 }}>
        {props.title}
      </Typography.Text>
      <ArrowRightOutlined style={{ color: 'var(--ant-color-text-quaternary)', fontSize: 12, flexShrink: 0 }} />
    </div>
  );
}

function QuickLinkGroups({ links }: { links: { title: string; link: string }[] }) {
  const { t } = useLocale();
  const byLink = new Map(links.map((item) => [item.link, item]));

  return (
    <Space direction="vertical" size={20} style={{ width: '100%' }}>
      {QUICK_LINK_GROUPS.map((group) => {
        const items = group.links.map((link) => byLink.get(link)).filter(Boolean) as {
          title: string;
          link: string;
        }[];
        if (!items.length) return null;

        return (
          <div key={group.labelKey}>
            <Typography.Text
              type="secondary"
              style={{ display: 'block', fontSize: 12, marginBottom: 10, fontWeight: 500 }}
            >
              {t(group.labelKey)}
            </Typography.Text>
            <Row gutter={[12, 12]}>
              {items.map((item) => (
                <Col xs={24} sm={12} md={8} lg={8} xl={8} key={item.link}>
                  <QuickLinkCard title={item.title} link={item.link} />
                </Col>
              ))}
            </Row>
          </div>
        );
      })}
    </Space>
  );
}

function buildKpiCards(summary: DashboardSummary, t: Translate): {
  title: string;
  value: number;
  link: string;
  intent?: MetricCardIntent;
  emptyHint?: string;
}[] {
  return [
    {
      title: t('dashboard.kpi.collectTasks'),
      value: summary.collectFailedCount ?? 0,
      link: '/collect/tasks',
      intent: 'data',
      emptyHint: t('dashboard.empty.noCollectTasks'),
    },
    {
      title: t('dashboard.kpi.drafts'),
      value: summary.draftTotal ?? summary.draftProducts + summary.readyProducts,
      link: '/product/drafts',
      emptyHint: t('dashboard.empty.noDrafts'),
    },
    {
      title: t('dashboard.kpi.aiReview'),
      value: (summary.aiPendingProducts ?? 0) + (summary.aiReplySuggestionPendingCount ?? 0),
      link: '/ai/operation-workbench',
      intent: 'ai',
      emptyHint: t('dashboard.empty.noAiReview'),
    },
    {
      title: t('dashboard.kpi.readinessIssues'),
      value: summary.readinessBlocked ?? summary.readinessBlockedProducts ?? 0,
      link: '/product/drafts?readiness=blocked',
      intent: 'warning',
      emptyHint: t('dashboard.empty.readinessPassed'),
    },
    {
      title: t('dashboard.kpi.publishFailures'),
      value: summary.publishFailedTasks ?? 0,
      link: '/product/publish-tasks?status=failed',
      intent: 'danger',
      emptyHint: t('dashboard.empty.noPublishFailures'),
    },
    {
      title: t('dashboard.kpi.orderExceptions'),
      value: summary.orderExceptions ?? summary.orderExceptionTotal ?? 0,
      link: '/orders/exceptions',
      intent: 'danger',
      emptyHint: t('dashboard.empty.noOrderExceptions'),
    },
    {
      title: t('dashboard.kpi.inventoryIssues'),
      value:
        (summary.inventoryAlerts ?? summary.lowStockSkus + summary.outOfStockSkus) +
        (summary.inventorySyncFailedCount ?? 0),
      link: '/inventory/alerts',
      intent: 'danger',
      emptyHint: t('dashboard.empty.inventoryHealthy'),
    },
    {
      title: t('dashboard.kpi.customerReplies'),
      value: summary.customerPendingReplyCount ?? summary.customerPendingConversations ?? 0,
      link: '/customer/conversations?status=pending_reply',
      intent: 'data',
      emptyHint: t('dashboard.empty.noCustomerReplies'),
    },
    {
      title: t('dashboard.kpi.failedTasks'),
      value: summary.failedTaskTotal ?? summary.failedTasks ?? 0,
      link: '/ops/task-center/failures',
      intent: 'danger',
      emptyHint: t('dashboard.empty.noFailedTasks'),
    },
    {
      title: t('dashboard.kpi.configurationRisks'),
      value: summary.configRiskCount ?? 0,
      link: '/settings/config-status',
      intent: 'warning',
      emptyHint: t('dashboard.empty.configurationHealthy'),
    },
  ];
}

function mergeRecentItems(
  recent: ProductOperationDashboard['recent'] | undefined,
  t: Translate,
): (DashboardRecentItem & { bucket: string })[] {
  if (!recent) return [];
  const buckets: { items: DashboardRecentItem[]; label: string }[] = [
    { items: recent.collectedProducts ?? [], label: t('dashboard.recent.collect') },
    { items: recent.aiTasks ?? [], label: t('dashboard.recent.aiTask') },
    { items: recent.imageTasks ?? [], label: t('dashboard.recent.imageTask') },
    { items: recent.publishTasks ?? [], label: t('dashboard.recent.productPublish') },
    { items: recent.failedTasks ?? [], label: t('dashboard.recent.failed') },
  ];
  return buckets
    .flatMap(({ items, label }) =>
      items.map((x) => ({
        ...x,
        bucket: t(RECENT_TYPE_KEYS[x.type] ?? 'dashboard.recent.other') || label,
      })),
    )
    .sort((a, b) => dayjs(b.occurredAt).valueOf() - dayjs(a.occurredAt).valueOf())
    .slice(0, 10);
}

const FUNNEL_STEP_META: Record<string, { icon: ReactNode; color: string; bg: string }> = {
  collected: { icon: <CloudUploadOutlined />, color: 'var(--ant-color-primary)', bg: 'var(--ant-color-primary-bg)' },
  draft: { icon: <FileTextOutlined />, color: 'var(--ant-color-primary)', bg: 'var(--ant-color-primary-bg)' },
  ai_text: { icon: <RobotOutlined />, color: 'var(--tm-ai-accent)', bg: 'var(--ant-color-primary-bg)' },
  ai_image: { icon: <PictureOutlined />, color: 'var(--ant-color-info)', bg: 'var(--ant-color-info-bg)' },
  readiness_pass: { icon: <SafetyCertificateOutlined />, color: 'var(--ant-color-success)', bg: 'var(--ant-color-success-bg)' },
  published: { icon: <CheckCircleOutlined />, color: 'var(--ant-color-success)', bg: 'var(--ant-color-success-bg)' },
};

function FunnelSteps({ steps }: { steps: DashboardFunnelStep[] }) {
  const list = steps.length ? steps : DEFAULT_FUNNEL;
  const topCount = list[0]?.count ?? 0;
  const max = Math.max(...list.map((s) => s.count ?? 0), 1);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {list.map((step, index) => {
        const count = step.count ?? 0;
        const meta = FUNNEL_STEP_META[step.key] ?? FUNNEL_STEP_META.collected;
        const barPct = count > 0 ? Math.max(8, Math.round((count / max) * 100)) : 0;
        const convPct = topCount > 0 ? Math.round((count / topCount) * 100) : 0;
        const isLast = index === list.length - 1;

        return (
          <div key={step.key} style={{ display: 'flex', gap: 14 }}>
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                width: 40,
                flexShrink: 0,
              }}
            >
              <div
                style={{
                  width: 40,
                  height: 40,
                  borderRadius: 12,
                  background: meta.bg,
                  color: meta.color,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: 18,
                  boxShadow: '0 0 0 1px var(--ant-color-border-secondary)',
                }}
              >
                {meta.icon}
              </div>
              {!isLast ? (
                <div
                  style={{
                    width: 2,
                    flex: 1,
                    minHeight: 20,
                    margin: '6px 0',
                    borderRadius: 1,
                    background: `linear-gradient(180deg, ${meta.color}, var(--ant-color-border-secondary))`,
                  }}
                />
              ) : null}
            </div>

            <div
              role="button"
              tabIndex={0}
              onClick={() => history.push(appendSourceToUrl(step.link))}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') history.push(appendSourceToUrl(step.link));
              }}
              style={{
                flex: 1,
                marginBottom: isLast ? 0 : 6,
                padding: '14px 16px',
                borderRadius: 10,
                border: '1px solid var(--ant-color-border-secondary)',
                background: count > 0 ? 'var(--ant-color-fill-quaternary)' : 'var(--ant-color-bg-container)',
                cursor: 'pointer',
                transition: 'border-color 0.2s, box-shadow 0.2s',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.borderColor = meta.color;
                e.currentTarget.style.boxShadow = 'var(--tm-shadow-card)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.borderColor = 'var(--ant-color-border-secondary)';
                e.currentTarget.style.boxShadow = 'none';
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: 12,
                  marginBottom: 10,
                }}
              >
                <Typography.Text strong style={{ fontSize: 14 }}>
                  {step.title}
                </Typography.Text>
                <Space size={8} align="center">
                  <Typography.Text
                    strong
                    style={{ fontSize: 18, color: count > 0 ? meta.color : undefined, lineHeight: 1 }}
                  >
                    {count}
                  </Typography.Text>
                  {index > 0 && topCount > 0 ? (
                    <Tag
                      bordered={false}
                      style={{
                        margin: 0,
                        background: meta.bg,
                        color: meta.color,
                        fontSize: 12,
                      }}
                    >
                      {convPct}%
                    </Tag>
                  ) : null}
                  <ArrowRightOutlined style={{ color: 'var(--ant-color-text-tertiary)', fontSize: 12 }} />
                </Space>
              </div>
              <div
                style={{
                  height: 10,
                  borderRadius: 999,
                  background: 'var(--ant-color-fill-secondary)',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    width: `${barPct}%`,
                    background: meta.color,
                    borderRadius: 999,
                    transition: 'width 0.45s ease',
                  }}
                />
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function DashboardSkeleton() {
  return (
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Skeleton active paragraph={{ rows: 2 }} />
      <Row gutter={[16, 16]}>
        {Array.from({ length: 6 }).map((_, i) => (
          <Col xs={24} sm={12} md={8} lg={8} xl={4} key={i}>
            <Skeleton active />
          </Col>
        ))}
      </Row>
      <Skeleton active paragraph={{ rows: 6 }} />
    </Space>
  );
}

export default function ProductOperationsDashboardPage() {
  const { t } = useLocale();
  const dashboardEmptyLocale = useListEmptyLocale('dashboard');
  const { state: urlState, setState: setUrlState, clearState: clearUrlState } =
    useUrlQueryState<Record<(typeof DASHBOARD_QUERY_KEYS)[number], string | undefined>>(
      DASHBOARD_QUERY_KEYS,
    );
  const [filters, setFilters] = useState<FilterState>(() => buildDashboardFiltersFromUrl(urlState));
  const [shops, setShops] = useState<ShopListRow[]>([]);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [board, setBoard] = useState<ProductOperationDashboard | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    void queryShops({ page: 1, pageSize: 200 })
      .then((res) => setShops(res?.list ?? []))
      .catch(() => setShops([]));
  }, []);

  useEffect(() => {
    setFilters(buildDashboardFiltersFromUrl(urlState));
  }, [
    urlState.end,
    urlState.platform,
    urlState.productSource,
    urlState.shopId,
    urlState.source,
    urlState.start,
  ]);

  useEffect(() => {
    const next = dashboardFiltersToUrlPatch(filters);
    if (sameDashboardUrlPatch(next, urlState)) return;
    setUrlState(next, { replace: true });
  }, [filters, setUrlState, urlState]);

  const queryParams = useMemo(() => {
    const [start, end] = filters.range ?? [];
    return {
      start: start ? start.startOf('day').toISOString() : undefined,
      end: end ? end.endOf('day').toISOString() : undefined,
      platform: filters.platform || undefined,
      shopId: filters.shopId || undefined,
      source: filters.source || undefined,
    };
  }, [filters]);

  const loadBoard = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await queryProductOperationDashboard(queryParams);
      setBoard(res ?? null);
    } catch (e) {
      setBoard(null);
      setError(e instanceof Error ? e : new Error(String(e ?? 'load_failed')));
    } finally {
      setLoading(false);
    }
  }, [queryParams]);

  useEffect(() => {
    void loadBoard();
  }, [loadBoard]);

  const summary = board?.summary ?? EMPTY_SUMMARY;
  const todos = useMemo(() => localizeDashboardItems(mergeTodos(board?.todos), t), [board?.todos, t]);
  const funnelSteps = useMemo(() => localizeDashboardItems(mergeFunnel(board?.funnel), t), [board?.funnel, t]);
  const exceptions = useMemo(() => localizeDashboardItems(mergeExceptions(board?.exceptions), t), [board?.exceptions, t]);
  const quickLinks = useMemo(() => localizeQuickLinks(DEFAULT_QUICK_LINKS, t), [t]);
  const recentFlat = useMemo(() => mergeRecentItems(board?.recent, t), [board?.recent, t]);
  const kpiCards = useMemo(() => buildKpiCards(summary, t), [summary, t]);
  const sources = useMemo(() => sourceOptions(t), [t]);

  const doRefresh = useCallback(() => {
    void loadBoard();
  }, [loadBoard]);

  useEffect(() => {
    if (!autoRefresh) return;
    const tick = () => {
      if (document.hidden) return;
      void loadBoard();
    };
    const id = window.setInterval(tick, 60_000);
    return () => window.clearInterval(id);
  }, [autoRefresh, loadBoard]);

  const welcomeActions: { label: string; icon: ReactNode; link: string }[] = [
    { label: t('dashboard.actions.collectProducts'), icon: <CloudUploadOutlined />, link: '/collect/hub' },
    { label: t('dashboard.actions.batchAi'), icon: <RobotOutlined />, link: '/ai/text-batches' },
    { label: t('dashboard.actions.aiImageTasks'), icon: <PictureOutlined />, link: '/ai/image-tasks' },
    { label: t('dashboard.actions.viewReadiness'), icon: <SafetyCertificateOutlined />, link: '/product/drafts?readiness=blocked' },
  ];

  return (
    <TmPageContainer
      title={t('dashboard.title')}
      subTitle={t('dashboard.description')}
      contentMaxWidth={layoutTokens.dashboardMaxWidth}
      extra={
        <OperationToolbar>
          <Button
            type={autoRefresh ? 'primary' : 'default'}
            ghost={autoRefresh}
            size="small"
            onClick={() => setAutoRefresh((v) => !v)}
          >
            {autoRefresh ? t('dashboard.actions.autoRefreshOn') : t('dashboard.actions.autoRefreshOff')}
          </Button>
          <Button icon={<ReloadOutlined />} onClick={doRefresh} loading={loading}>
            {t('dashboard.actions.reload')}
          </Button>
        </OperationToolbar>
      }
    >
      {/* Filters */}
      <ProCard variant="outlined" style={{ marginBottom: 16 }} bodyStyle={{ padding: '12px 16px' }}>
        <Space wrap size={[12, 12]}>
          <RangePicker
            value={filters.range}
            onChange={(v) => setFilters((f) => ({ ...f, range: v as [Dayjs, Dayjs] | undefined }))}
            allowClear
            placeholder={[t('dashboard.filters.startDate'), t('dashboard.filters.endDate')]}
          />
          <Select
            allowClear
            placeholder={t('dashboard.filters.platform')}
            style={{ width: 140 }}
            options={PLATFORM_OPTIONS}
            value={filters.platform}
            onChange={(v) => setFilters((f) => ({ ...f, platform: v }))}
          />
          <Select
            allowClear
            placeholder={t('dashboard.filters.shop')}
            style={{ width: 180 }}
            showSearch
            optionFilterProp="label"
            options={shops.map((s) => ({
              label: s.shopName || s.id,
              value: s.id,
            }))}
            value={filters.shopId}
            onChange={(v) => setFilters((f) => ({ ...f, shopId: v }))}
          />
          <Select
            allowClear
            placeholder={t('dashboard.filters.productSource')}
            style={{ width: 140 }}
            options={sources}
            value={filters.source}
            onChange={(v) => setFilters((f) => ({ ...f, source: v }))}
          />
          <Button
            onClick={() => {
              setFilters({
                range: undefined,
                platform: undefined,
                shopId: undefined,
                source: undefined,
              });
              clearUrlState(DASHBOARD_QUERY_KEYS, { replace: true });
            }}
          >
            {t('dashboard.actions.resetFilters')}
          </Button>
        </Space>
      </ProCard>

      {error ? (
        <Result
          status="error"
          title={t('dashboard.errors.loadFailed')}
          subTitle={error instanceof Error ? error.message : t('dashboard.errors.network')}
          extra={
            <Button type="primary" onClick={doRefresh}>
              {t('dashboard.actions.reload')}
            </Button>
          }
        />
      ) : loading && !board ? (
        <DashboardSkeleton />
      ) : (
        <>
          {/* 1. Top welcome area + KPI */}
          <ProCard variant="outlined" style={{ marginBottom: 16 }} bodyStyle={{ padding: '20px 24px' }}>
            <Row align="middle" gutter={[16, 16]} wrap style={{ marginBottom: 20 }}>
              <Col flex="auto">
                <Typography.Title level={4} style={{ margin: 0 }}>
                  {t('dashboard.sections.todayOverview')}
                </Typography.Title>
              </Col>
              <Col>
                <OperationToolbar>
                  {welcomeActions.map((a) => (
                    <Button key={a.link} icon={a.icon} onClick={() => history.push(appendSourceToUrl(a.link))}>
                      {a.label}
                    </Button>
                  ))}
                </OperationToolbar>
              </Col>
            </Row>
            <Row gutter={[12, 12]}>
              {kpiCards.map((card) => (
                <Col xs={12} sm={8} md={6} lg={4} xl={4} key={card.title}>
                  <MetricCard
                    title={card.title}
                    value={card.value}
                    description={card.value > 0 ? undefined : card.emptyHint}
                    intent={card.intent}
                    onClick={() => history.push(appendSourceToUrl(card.link))}
                  />
                </Col>
              ))}
            </Row>
          </ProCard>

          {/* 2. Today's to-dos */}
          <ProCard title={t('dashboard.sections.todayTodos')} variant="outlined" style={{ marginBottom: 16 }}>
            <Row gutter={[16, 16]}>
              {todos.map((item) => (
                <Col xs={24} sm={12} md={8} lg={6} xl={6} key={item.key || item.id}>
                  <TodoCardItem item={item} />
                </Col>
              ))}
            </Row>
          </ProCard>

          <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
            {/* 4. AI product operations progress funnel */}
            <Col xs={24} lg={10}>
              <ProCard title={t('dashboard.sections.aiProgress')} variant="outlined" bodyStyle={{ padding: '16px 20px 12px' }}>
                <FunnelSteps steps={funnelSteps} />
              </ProCard>
            </Col>

            {/* 5. Exception and failure alerts */}
            <Col xs={24} lg={14}>
              <ProCard title={t('dashboard.sections.exceptions')} variant="outlined">
                <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                  {exceptions.map((item) => (
                    <ExceptionRow key={item.key} item={item} />
                  ))}
                </div>
              </ProCard>
            </Col>
          </Row>

          {/* Recent activity */}
          <ProCard title={t('dashboard.sections.recentActivity')} variant="outlined" style={{ marginBottom: 16 }} bodyStyle={{ padding: '12px 16px 16px' }}>
            {recentFlat.length ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                {recentFlat.map((item, idx) => (
                  <RecentActivityRow
                    key={`${item.type}-${item.occurredAt}-${idx}`}
                    item={item}
                    bucket={item.bucket}
                  />
                ))}
              </div>
            ) : (
              dashboardEmptyLocale.emptyText
            )}
          </ProCard>

          {/* Quick links */}
          <ProCard title={t('dashboard.sections.quickLinks')} variant="outlined" bodyStyle={{ padding: '16px 20px 20px' }}>
            <QuickLinkGroups links={quickLinks} />
          </ProCard>
        </>
      )}
    </TmPageContainer>
  );
}

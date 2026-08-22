import { TmPageContainer } from '@/components/ui';
import { CUSTOMER_CONVERSATION_STATUS } from '@/constants/status';
import { PLATFORM_OPTIONS } from '@/constants/userFriendly';
import { getCustomerDashboard, type CustomerDashboardSummary } from '@/services/customer';
import { queryShops } from '@/services/shops';
import { useListEmptyLocale } from '@/hooks/useListEmptyLocale';
import { useUrlQueryState } from '@/hooks/useUrlState';
import { appendSourceToUrl } from '@/utils/urlState';
import { history } from '@umijs/max';
import { Alert, Button, Card, Col, Row, Select, Space, Spin, Statistic, Tag } from 'antd';
import { useEffect, useState } from 'react';

const HUB_QUERY_KEYS = ['platform', 'shopId', 'source'] as const;

export default function CustomerHubPage() {
  const emptyLocale = useListEmptyLocale('customerHub', { permissionScoped: true });
  const { state: urlState, setState: setUrlState } =
    useUrlQueryState<Record<(typeof HUB_QUERY_KEYS)[number], string | undefined>>(HUB_QUERY_KEYS);
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<CustomerDashboardSummary | null>(null);
  const [shopOptions, setShopOptions] = useState<{ label: string; value: string }[]>([]);

  useEffect(() => {
    void (async () => {
      setLoading(true);
      try {
        setData(await getCustomerDashboard());
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  useEffect(() => {
    void (async () => {
      try {
        const res = await queryShops({ page: 1, pageSize: 500 });
        setShopOptions(
          res.list.map((s) => ({
            label: `${s.shopName} (${s.platform})`,
            value: s.id,
          })),
        );
      } catch {
        setShopOptions([]);
      }
    })();
  }, []);

  const buildConversationLink = (extraQuery: string) => {
    const sp = new URLSearchParams(extraQuery);
    if (urlState.platform) sp.set('platform', urlState.platform);
    if (urlState.shopId) sp.set('shopId', urlState.shopId);
    const qs = sp.toString();
    return appendSourceToUrl(qs ? `/customer/conversations?${qs}` : '/customer/conversations');
  };

  return (
    <TmPageContainer
      title="Customer service"
      subTitle="View conversations awaiting replies, AI suggestions, controlled auto-replies, and message-sync status."
    >
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="Manual confirmation is the default; auto-replies must be explicitly enabled"
        description="By default, only AI suggestions are generated. Administrators can enable low-risk auto-replies by shop; high-risk content always goes to a human."
      />
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space wrap>
          <Select
            allowClear
            placeholder="Platform"
            style={{ width: 160 }}
            options={PLATFORM_OPTIONS}
            value={urlState.platform}
            onChange={(v) => setUrlState({ platform: v || undefined }, { replace: true })}
          />
          <Select
            allowClear
            placeholder="Shop"
            style={{ width: 220 }}
            showSearch
            optionFilterProp="label"
            options={shopOptions}
            value={urlState.shopId}
            onChange={(v) => setUrlState({ shopId: v || undefined }, { replace: true })}
          />
        </Space>
      </Card>
      <Spin spinning={loading}>
        {data ? (
          <>
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={12} md={8} lg={6}>
                <Card hoverable onClick={() => history.push(buildConversationLink('replyStatus=pending'))}>
                  <Statistic title="Conversations awaiting reply" value={data.pendingReplyCount} />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8} lg={6}>
                <Card>
                  <Statistic title="New messages today" value={data.todayNewMessages} />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8} lg={6}>
                <Card hoverable onClick={() => history.push(buildConversationLink('aiSuggestionStatus=pending'))}>
                  <Statistic title="AI suggestions awaiting confirmation" value={data.aiSuggestionPendingCount} />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8} lg={6}>
                <Card hoverable onClick={() => history.push(buildConversationLink('sendStatus=failed'))}>
                  <Statistic title="Send failures" value={data.sendFailureCount} valueStyle={{ color: data.sendFailureCount ? 'var(--ant-color-error)' : undefined }} />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8} lg={6}>
                <Card>
                  <Statistic title="Unauthorized shops" value={data.unauthorizedShopCount} />
                  {data.unauthorizedShopCount > 0 ? (
                    <Button type="link" size="small" onClick={() => history.push('/settings/platforms')}>
                      Open platform authorization
                    </Button>
                  ) : null}
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8} lg={6}>
                <Card hoverable onClick={() => history.push(appendSourceToUrl('/customer/message-sync-tasks'))}>
                  <Statistic title="Sync-task failures (7 days)" value={data.syncTaskFailureCount} />
                </Card>
              </Col>
            </Row>
            <Card title="Quick links" style={{ marginTop: 16 }}>
              <Row gutter={[8, 8]}>
                <Col>
                  <Button type="primary" onClick={() => history.push(buildConversationLink(''))}>
                    Conversations
                  </Button>
                </Col>
                <Col>
                  <Button onClick={() => history.push('/customer/message-sync-tasks')}>Message sync tasks</Button>
                </Col>
                <Col>
                  <Button onClick={() => history.push('/customer/auto-reply-settings')}>AI auto-replies</Button>
                </Col>
                <Col>
                  <Button onClick={() => history.push(appendSourceToUrl('/ops/task-center/failures?taskType=customer_failure', 'taskcenter'))}>
                    Failed-task center
                  </Button>
                </Col>
              </Row>
            </Card>
            <Card title="Conversation status guide" style={{ marginTop: 16 }} size="small">
              {Object.entries(CUSTOMER_CONVERSATION_STATUS).map(([k, v]) => (
                <Tag key={k} color={v.color} style={{ marginBottom: 4 }}>
                  {v.text}
                </Tag>
              ))}
            </Card>
          </>
        ) : (
          emptyLocale.emptyText
        )}
      </Spin>
    </TmPageContainer>
  );
}

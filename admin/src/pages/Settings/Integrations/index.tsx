import { Link } from '@umijs/renderer-react';
import {
  ApiOutlined,
  CloudOutlined,
  MailOutlined,
  PictureOutlined,
  RightOutlined,
  RobotOutlined,
} from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import { TmPageContainer } from '@/components/ui';
import { Col, Row, Space, Spin, Statistic, Tag, Typography } from 'antd';
import type { ComponentType, CSSProperties, ReactNode } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { PAGE_COPY } from '@/constants/copywriting';
import {
  aiTextProviderLabel,
  imageProviderLabel,
  imageSubServiceStatusLabel,
  integrationConfiguredTag,
  storageKindLabel,
} from '@/constants/integrations';
import { PLATFORM_PROVIDER_STATUS } from '@/constants/status';
import { preferredPlatformTabOrder } from '@/services/platformOpen';
import { fetchIntegrationsOverview, type IntegrationOverviewData } from '@/services/settings';

const { Text, Paragraph, Title } = Typography;

type HubCardProps = {
  title: string;
  desc: string;
  configured: boolean;
  to: string;
  Icon: ComponentType<{ style?: CSSProperties }>;
  extra?: ReactNode;
};

function IntegrationHubCard({ title, desc, configured, to, Icon, extra }: HubCardProps) {
  const tag = integrationConfiguredTag(configured);
  return (
    <div
      style={{
        height: '100%',
        padding: '20px 20px 16px',
        borderRadius: 8,
        border: '1px solid var(--ant-color-border-secondary)',
        background: 'var(--ant-color-bg-container)',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <Space align="start" size="middle" style={{ marginBottom: 12 }}>
        <div
          style={{
            width: 40,
            height: 40,
            borderRadius: 8,
            background: 'linear-gradient(135deg, var(--ant-color-primary-bg) 0%, var(--ant-color-info-bg) 100%)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}
        >
          <Icon style={{ fontSize: 20, color: 'var(--ant-color-primary)' }} />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <Space wrap size={[8, 4]}>
            <Title level={5} style={{ margin: 0 }}>
              {title}
            </Title>
            <Tag color={tag.color}>{tag.text}</Tag>
          </Space>
          <Paragraph type="secondary" style={{ margin: '6px 0 0', fontSize: 13 }}>
            {desc}
          </Paragraph>
        </div>
      </Space>
      {extra ? <div style={{ flex: 1, marginBottom: 12 }}>{extra}</div> : <div style={{ flex: 1 }} />}
      <Link to={to} style={{ fontSize: 13 }}>
        Configure <RightOutlined style={{ fontSize: 11 }} />
      </Link>
    </div>
  );
}

type PlatformCardProps = {
  name: string;
  appConfigured: boolean;
  status: string;
};

function PlatformIntegrationCard({ name, appConfigured, status }: PlatformCardProps) {
  const cfg = integrationConfiguredTag(appConfigured);
  const runtime = PLATFORM_PROVIDER_STATUS[status as keyof typeof PLATFORM_PROVIDER_STATUS];
  return (
    <div
      style={{
        height: '100%',
        padding: '16px 16px 14px',
        borderRadius: 8,
        border: '1px solid var(--ant-color-border-secondary)',
        background: 'var(--ant-color-bg-container)',
      }}
    >
      <Space wrap size={[8, 4]} style={{ marginBottom: 8 }}>
        <Text strong>{name}</Text>
        <Tag color={cfg.color}>{cfg.text}</Tag>
      </Space>
      <div style={{ marginBottom: 10 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          Runtime{' '}
        </Text>
        <Tag color={runtime?.color}>{runtime?.text ?? status}</Tag>
      </div>
      <Link to="/settings/platforms" style={{ fontSize: 13 }}>
        Edit app settings <RightOutlined style={{ fontSize: 11 }} />
      </Link>
    </div>
  );
}

export default function IntegrationsHubPage() {
  const [loading, setLoading] = useState(true);
  const [data, setData] = useState<IntegrationOverviewData | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const row = await fetchIntegrationsOverview();
      setData(row);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const sortedPlatforms = useMemo(() => {
    if (!data?.platforms?.length) return [];
    return [...data.platforms].sort(
      (a, b) => preferredPlatformTabOrder(a.platform) - preferredPlatformTabOrder(b.platform) || a.name.localeCompare(b.name),
    );
  }, [data?.platforms]);

  const summary = useMemo(() => {
    if (!data) return { configured: 0, total: 0 };
    const core = [data.ai.configured, data.storage.configured, data.mail.configured];
    const imageReady = data.image.removebg || data.image.openaiImage || data.image.comfyui;
    core.push(!!imageReady);
    const configured = core.filter(Boolean).length;
    return { configured, total: core.length };
  }, [data]);

  const aiDetail = useMemo(() => {
    if (!data) return null;
    const parts: string[] = [];
    if (data.ai.provider) {
      parts.push(`Provider: ${aiTextProviderLabel(data.ai.provider)}`);
    }
    if (data.ai.model) {
      parts.push(`Model: ${data.ai.model}`);
    }
    return parts.length ? parts.join(' · ') : 'API key and endpoint are not configured';
  }, [data]);

  const imageDetail = useMemo(() => {
    if (!data) return null;
    const items = [
      { label: 'remove.bg background removal', ok: data.image.removebg, kind: 'key' as const },
      { label: 'OpenAI image generation', ok: data.image.openaiImage, kind: 'key' as const },
      { label: 'ComfyUI workflow', ok: data.image.comfyui, kind: 'url' as const },
    ];
    return (
      <Space direction="vertical" size={6} style={{ width: '100%' }}>
        {data.image.providerCurrent ? (
          <Text type="secondary" style={{ fontSize: 12 }}>
            Current default: {imageProviderLabel(data.image.providerCurrent)}
          </Text>
        ) : null}
        <Space wrap size={[4, 6]}>
          {items.map((item) => (
            <Tag key={item.label} color={item.ok ? 'success' : 'default'}>
              {item.label} · {imageSubServiceStatusLabel(item.ok, item.kind)}
            </Tag>
          ))}
        </Space>
      </Space>
    );
  }, [data]);

  const storageDetail = useMemo(() => {
    if (!data) return null;
    const kindLabel = storageKindLabel(data.storage.kind);
    return data.storage.configured ? `Current method: ${kindLabel}` : `Complete the credentials required by ${kindLabel}`;
  }, [data]);

  return (
    <TmPageContainer
      title={PAGE_COPY.integrations.title}
      subTitle={PAGE_COPY.integrations.description}
    >
      <Spin spinning={loading}>
        {data ? (
          <>
            <ProCard variant="outlined" style={{ marginBottom: 16 }}>
              <Row gutter={[24, 16]}>
                <Col xs={24} sm={12} md={8}>
                  <Statistic title="Core integrations ready" value={`${summary.configured} / ${summary.total}`} />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Statistic title="Platform app settings" value={sortedPlatforms.filter((p) => p.appConfigured).length} suffix={`/ ${sortedPlatforms.length}`} />
                </Col>
                <Col xs={24} sm={12} md={8}>
                  <Statistic
                    title="Custom collection rules"
                    value={data.collectRulesCount}
                    suffix={
                      <Link to="/collect/rules" style={{ fontSize: 13, marginLeft: 8 }}>
                        Manage rules
                      </Link>
                    }
                  />
                </Col>
              </Row>
            </ProCard>

            <Title level={5} style={{ marginBottom: 12 }}>
              Core capabilities
            </Title>
            <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
              <Col xs={24} sm={12} lg={8}>
                <IntegrationHubCard
                  title="AI text"
                  desc="Text capabilities such as title optimization, description generation, and customer-service suggestions"
                  configured={data.ai.configured}
                  to="/settings/ai"
                  Icon={RobotOutlined}
                  extra={
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {aiDetail}
                    </Text>
                  }
                />
              </Col>
              <Col xs={24} sm={12} lg={8}>
                <IntegrationHubCard
                  title="Image AI"
                  desc="Image processing such as background removal, scene images, and ComfyUI workflows"
                  configured={data.image.removebg || data.image.openaiImage || data.image.comfyui}
                  to="/settings/image"
                  Icon={PictureOutlined}
                  extra={imageDetail}
                />
              </Col>
              <Col xs={24} sm={12} lg={8}>
                <IntegrationHubCard
                  title="File storage"
                  desc="Uploading and accessing product images and attachments"
                  configured={data.storage.configured}
                  to="/settings/storage"
                  Icon={CloudOutlined}
                  extra={
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {storageDetail}
                    </Text>
                  }
                />
              </Col>
              <Col xs={24} sm={12} lg={8}>
                <IntegrationHubCard
                  title="Email SMTP"
                  desc="System email such as alert notifications and test messages"
                  configured={data.mail.configured}
                  to="/settings/email"
                  Icon={MailOutlined}
                  extra={
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {data.mail.configured ? 'SMTP host and sender address are configured' : 'Configure the SMTP host and sender address'}
                    </Text>
                  }
                />
              </Col>
              <Col xs={24} sm={12} lg={8}>
                <IntegrationHubCard
                  title="Custom collection rules"
                  desc="XPath/CSS collection rules for sites without a built-in collector"
                  configured={data.collectRulesCount > 0}
                  to="/collect/rules"
                  Icon={ApiOutlined}
                  extra={
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {data.collectRulesCount} rules created
                    </Text>
                  }
                />
              </Col>
            </Row>

            {sortedPlatforms.length ? (
              <>
                <Title level={5} style={{ marginBottom: 4 }}>
                  Cross-border platforms
                </Title>
                <Paragraph type="secondary" style={{ marginBottom: 12 }}>
                  These are open-platform app settings for each platform (app key, secret, and similar values). Credentials created by shop authorization are stored under Shops → Authorization and must not be entered here.
                </Paragraph>
                <Row gutter={[16, 16]}>
                  {sortedPlatforms.map((p) => (
                    <Col xs={24} sm={12} md={8} lg={6} key={p.platform}>
                      <PlatformIntegrationCard
                        name={p.name}
                        appConfigured={p.appConfigured}
                        status={p.status}
                      />
                    </Col>
                  ))}
                </Row>
                <div style={{ marginTop: 16 }}>
                  <Link to="/settings/platforms">
                    Open Platform settings <RightOutlined style={{ fontSize: 11 }} />
                  </Link>
                </div>
              </>
            ) : null}
          </>
        ) : null}
      </Spin>
    </TmPageContainer>
  );
}

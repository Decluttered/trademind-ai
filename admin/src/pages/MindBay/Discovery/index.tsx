import { useState } from 'react';
import { Alert, Button, Form, Input, Space, Typography } from 'antd';
import { ErrorAlert, SectionCard, TmPageContainer } from '@/components/ui';
import { useLocale } from '@/locale';
import { runDiscovery, type ProductSnapshot } from '@/services/mindbay';

const { Text } = Typography;

export default function MindBayDiscoveryPage() {
  const { t } = useLocale();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [snapshot, setSnapshot] = useState<ProductSnapshot>();

  const submit = async ({ url }: { url: string }) => {
    setLoading(true);
    setError('');
    try {
      const out = await runDiscovery(url);
      setSnapshot(out.snapshot);
    } catch (e) {
      setError((e as Error).message || t('mindbay.discovery.fail'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <TmPageContainer title={t('mindbay.discovery.title')} subTitle={t('mindbay.discovery.subTitle')}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {error ? <ErrorAlert title={t('mindbay.discovery.errorTitle')} actionHint={error} /> : null}
        <SectionCard title={t('mindbay.discovery.cardTitle')} description={t('mindbay.discovery.cardDescription')}>
          <Form layout="vertical" onFinish={submit}>
            <Form.Item
              name="url"
              label={t('mindbay.discovery.urlLabel')}
              rules={[
                { required: true, message: t('mindbay.discovery.urlRequired') },
                {
                  pattern: /^https:\/\/(?:www\.)?amazon\.de\/(?:dp|gp\/product)\/[A-Z0-9]{10}/i,
                  message: t('mindbay.discovery.urlInvalid'),
                },
              ]}
            >
              <Input placeholder="https://www.amazon.de/dp/B012345678" />
            </Form.Item>
            <Button type="primary" htmlType="submit" loading={loading}>
              {t('mindbay.discovery.submit')}
            </Button>
          </Form>
        </SectionCard>
        {snapshot ? (
          <Alert
            type="success"
            showIcon
            message={t('mindbay.discovery.success')}
            description={
              <Text>
                {snapshot.title} · {snapshot.id}
              </Text>
            }
          />
        ) : null}
      </Space>
    </TmPageContainer>
  );
}

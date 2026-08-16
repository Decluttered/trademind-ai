import { useCallback, useEffect, useMemo, useState } from 'react';
import type { ProColumns } from '@ant-design/pro-components';
import { Col, Row, Space, Tag } from 'antd';
import { ErrorAlert, MetricCard, TmPageContainer, TmProTable } from '@/components/ui';
import { useLocale } from '@/locale';
import { formatEuroCents, getProfitReport, type ProfitLedgerEntry } from '@/services/mindbay';

const labelDe: Record<ProfitLedgerEntry['entryType'], string> = {
  expected_revenue: 'Erwarteter Umsatz',
  expected_cost: 'Erwartete Kosten',
  expected_fees: 'Erwartete Gebühren',
  expected_margin: 'Erwartete Marge',
  actual_revenue: 'Ist-Umsatz',
  actual_supplier_cost: 'Ist-Lieferantenkosten',
  actual_fees: 'Ist-Gebühren',
  shipping_cost: 'Versandkosten',
  refund_cost: 'Erstattungskosten',
  realized_profit: 'Realisierter Gewinn',
};

const labelEn: Record<ProfitLedgerEntry['entryType'], string> = {
  expected_revenue: 'Expected revenue',
  expected_cost: 'Expected cost',
  expected_fees: 'Expected fees',
  expected_margin: 'Expected margin',
  actual_revenue: 'Actual revenue',
  actual_supplier_cost: 'Actual supplier cost',
  actual_fees: 'Actual fees',
  shipping_cost: 'Shipping cost',
  refund_cost: 'Refund cost',
  realized_profit: 'Realized profit',
};

export default function ProfitPage() {
  const { t, locale } = useLocale();
  const entryLabels = locale === 'de' ? labelDe : locale === 'zh' ? labelDe : labelEn;
  const [items, setItems] = useState<ProfitLedgerEntry[]>([]);
  const [totals, setTotals] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const report = await getProfitReport();
      setItems(report.items || []);
      setTotals(report.totals || {});
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const columns = useMemo<ProColumns<ProfitLedgerEntry>[]>(
    () => [
      { title: locale === 'en' ? 'Time' : 'Zeitpunkt', dataIndex: 'occurredAt', width: 180, valueType: 'dateTime' },
      {
        title: 'Phase',
        dataIndex: 'phase',
        width: 120,
        render: (_, row) => (
          <Tag color={row.phase === 'actual' ? 'green' : 'blue'}>
            {row.phase === 'actual'
              ? locale === 'en'
                ? 'realized'
                : 'realisiert'
              : locale === 'en'
                ? 'forecast'
                : 'prognostiziert'}
          </Tag>
        ),
      },
      { title: locale === 'en' ? 'Type' : 'Typ', dataIndex: 'entryType', width: 190, render: (_, row) => entryLabels[row.entryType] },
      { title: locale === 'en' ? 'Amount' : 'Betrag', width: 140, render: (_, row) => formatEuroCents(row.amountCents) },
      { title: locale === 'en' ? 'Rule' : 'Berechnungsregel', dataIndex: 'calculationRule', ellipsis: true },
      { title: locale === 'en' ? 'Evidence' : 'Quellenbeleg', dataIndex: 'evidenceId', width: 220, copyable: true, ellipsis: true },
    ],
    [entryLabels, locale],
  );

  return (
    <TmPageContainer title={t('mindbay.profit.title')} subTitle={t('mindbay.profit.subTitle')}>
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {error ? <ErrorAlert title={t('mindbay.profit.errorTitle')} actionHint={error} /> : null}
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} xl={6}>
            <MetricCard title={t('mindbay.profit.expectedRevenue')} value={formatEuroCents(totals.expected_revenue)} intent="primary" />
          </Col>
          <Col xs={24} sm={12} xl={6}>
            <MetricCard title={t('mindbay.profit.expectedCost')} value={formatEuroCents(totals.expected_cost)} intent="warning" />
          </Col>
          <Col xs={24} sm={12} xl={6}>
            <MetricCard title={t('mindbay.profit.expectedMargin')} value={formatEuroCents(totals.expected_margin)} intent="success" />
          </Col>
          <Col xs={24} sm={12} xl={6}>
            <MetricCard
              title={t('mindbay.profit.realizedProfit')}
              value={formatEuroCents(totals.realized_profit)}
              description={t('mindbay.profit.realizedHint')}
              intent="data"
            />
          </Col>
        </Row>
        <TmProTable
          rowKey="id"
          headerTitle={t('mindbay.profit.entries')}
          loading={loading}
          search={false}
          options={false}
          pagination={false}
          dataSource={items}
          columns={columns}
          scroll={{ x: 1000 }}
          locale={{ emptyText: t('mindbay.profit.empty') }}
        />
      </Space>
    </TmPageContainer>
  );
}

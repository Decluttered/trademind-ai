import { useCallback, useEffect, useMemo, useState } from 'react';
import type { ProColumns } from '@ant-design/pro-components';
import { Col, Row, Space, Tag } from 'antd';
import { ErrorAlert, MetricCard, TmPageContainer, TmProTable } from '@/components/ui';
import { formatEuroCents, getProfitReport, type ProfitLedgerEntry } from '@/services/mindbay';

const label: Record<ProfitLedgerEntry['entryType'], string> = { expected_revenue: 'Erwarteter Umsatz', expected_cost: 'Erwartete Kosten', expected_fees: 'Erwartete Gebühren', expected_margin: 'Erwartete Marge', actual_revenue: 'Ist-Umsatz', actual_supplier_cost: 'Ist-Lieferantenkosten', actual_fees: 'Ist-Gebühren', shipping_cost: 'Versandkosten', refund_cost: 'Erstattungskosten', realized_profit: 'Realisierter Gewinn' };

export default function ProfitPage() {
  const [items, setItems] = useState<ProfitLedgerEntry[]>([]);
  const [totals, setTotals] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const load = useCallback(async () => { setLoading(true); setError(''); try { const report = await getProfitReport(); setItems(report.items || []); setTotals(report.totals || {}); } catch (cause) { setError((cause as Error).message); } finally { setLoading(false); } }, []);
  useEffect(() => { void load(); }, [load]);
  const columns = useMemo<ProColumns<ProfitLedgerEntry>[]>(() => [
    { title: 'Zeitpunkt', dataIndex: 'occurredAt', width: 180, valueType: 'dateTime' },
    { title: 'Phase', dataIndex: 'phase', width: 120, render: (_, row) => <Tag color={row.phase === 'actual' ? 'green' : 'blue'}>{row.phase === 'actual' ? 'realisiert' : 'prognostiziert'}</Tag> },
    { title: 'Typ', dataIndex: 'entryType', width: 190, render: (_, row) => label[row.entryType] },
    { title: 'Betrag', width: 140, render: (_, row) => formatEuroCents(row.amountCents) },
    { title: 'Berechnungsregel', dataIndex: 'calculationRule', ellipsis: true },
    { title: 'Quellenbeleg', dataIndex: 'evidenceId', width: 220, copyable: true, ellipsis: true },
  ], []);
  return <TmPageContainer title="Profit Ledger" subTitle="Erwartete und realisierte Beträge bleiben als append-only Cent-Buchungen nachvollziehbar getrennt.">
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {error ? <ErrorAlert title="Profit-Bericht konnte nicht geladen werden" actionHint={error} /> : null}
      <Row gutter={[16, 16]}><Col xs={24} sm={12} xl={6}><MetricCard title="Erwarteter Umsatz" value={formatEuroCents(totals.expected_revenue)} intent="primary" /></Col><Col xs={24} sm={12} xl={6}><MetricCard title="Erwartete Kosten" value={formatEuroCents(totals.expected_cost)} intent="warning" /></Col><Col xs={24} sm={12} xl={6}><MetricCard title="Erwartete Marge" value={formatEuroCents(totals.expected_margin)} intent="success" /></Col><Col xs={24} sm={12} xl={6}><MetricCard title="Realisierter Gewinn" value={formatEuroCents(totals.realized_profit)} description="Wird ab Phase 4 mit Sales gefüllt." intent="data" /></Col></Row>
      <TmProTable rowKey="id" headerTitle="Ledger-Buchungen" loading={loading} search={false} options={false} pagination={false} dataSource={items} columns={columns} scroll={{ x: 1000 }} locale={{ emptyText: 'Noch keine Ledger-Buchungen. Ein Monitoring-Lauf erzeugt erwartete Werte.' }} />
    </Space>
  </TmPageContainer>;
}

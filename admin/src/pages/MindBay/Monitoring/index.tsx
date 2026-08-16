import { useCallback, useEffect, useMemo, useState } from 'react';
import type { ProColumns } from '@ant-design/pro-components';
import { Alert, Button, Form, Input, InputNumber, Modal, Select, Space, Switch, Tag, message } from 'antd';
import { ErrorAlert, SectionCard, TmPageContainer, TmProTable } from '@/components/ui';
import { usePermission } from '@/hooks/usePermission';
import { PERMISSIONS } from '@/utils/permission';
import {
  applyPriceDecision,
  createPriceRule,
  formatEuroCents,
  listMonitorableListings,
  listPriceDecisions,
  listPriceRules,
  runMonitor,
  type MarketplaceListing,
  type PriceDecision,
  type PriceRule,
} from '@/services/mindbay';
import { useLocale } from '@/locale';

type RuleForm = {
  name: string;
  minMarginPercent: number;
  targetMarginPercent: number;
  maxPriceCents?: number;
  maxDeltaCents: number;
  maxDeltaPercent: number;
  cooldownMinutes: number;
  platformFeePercent: number;
  shippingCents: number;
  reserveCents: number;
  autoApply: boolean;
};

const outcomeColor: Record<PriceDecision['outcome'], string> = { NO_CHANGE: 'default', PROPOSED: 'blue', AUTO_APPLIED: 'green', BLOCKED_MARGIN: 'red', BLOCKED_POLICY: 'orange', BLOCKED_COOLDOWN: 'gold' };

export default function MonitoringPage() {
  const { t } = useLocale();
  const { readonly, can } = usePermission();
  const writable = !readonly && can(PERMISSIONS.PRODUCT_WRITE);
  const [listings, setListings] = useState<MarketplaceListing[]>([]);
  const [rules, setRules] = useState<PriceRule[]>([]);
  const [decisions, setDecisions] = useState<PriceDecision[]>([]);
  const [selectedListing, setSelectedListing] = useState<string>();
  const [selectedRule, setSelectedRule] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState('');
  const [error, setError] = useState('');
  const [runOpen, setRunOpen] = useState(false);
  const [ruleOpen, setRuleOpen] = useState(false);
  const [applyDecision, setApplyDecision] = useState<PriceDecision>();
  const [ruleForm] = Form.useForm<RuleForm>();

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [listingResult, ruleResult, decisionResult] = await Promise.all([listMonitorableListings(), listPriceRules(), listPriceDecisions()]);
      setListings(listingResult.items || []);
      setRules(ruleResult.items || []);
      setDecisions(decisionResult.items || []);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => { void load(); }, [load]);

  const createRule = async () => {
    const value = await ruleForm.validateFields();
    setAction('rule'); setError('');
    try {
      await createPriceRule({ name: value.name, minMarginBasisPoints: Math.round(value.minMarginPercent * 100), targetMarginBasisPoints: Math.round(value.targetMarginPercent * 100), maxPriceCents: value.maxPriceCents, maxDeltaCents: value.maxDeltaCents, maxDeltaBasisPoints: Math.round(value.maxDeltaPercent * 100), cooldownSeconds: Math.round(value.cooldownMinutes * 60), platformFeeBasisPoints: Math.round(value.platformFeePercent * 100), shippingCents: value.shippingCents, reserveCents: value.reserveCents, autoApply: value.autoApply });
      message.success('Neue Preisregel-Version gespeichert.');
      setRuleOpen(false); ruleForm.resetFields(); await load();
    } catch (cause) { setError((cause as Error).message); } finally { setAction(''); }
  };
  const run = async () => {
    if (!selectedListing || !selectedRule) return;
    setAction('run'); setError('');
    try { await runMonitor({ marketplaceListingId: selectedListing, priceRuleId: selectedRule, trigger: 'manual' }); message.success('Monitoring-Lauf abgeschlossen; Entscheidung wurde versioniert.'); setRunOpen(false); await load(); }
    catch (cause) { setError((cause as Error).message); } finally { setAction(''); }
  };
  const apply = async () => {
    if (!applyDecision) return;
    setAction('apply'); setError('');
    try { await applyPriceDecision(applyDecision.id); message.success('Preisentscheidung verarbeitet. DRY_RUN verändert eBay nicht.'); setApplyDecision(undefined); await load(); }
    catch (cause) { setError((cause as Error).message); } finally { setAction(''); }
  };

  const columns = useMemo<ProColumns<PriceDecision>[]>(() => [
    { title: 'Erstellt', dataIndex: 'createdAt', width: 180, valueType: 'dateTime' },
    { title: 'Outcome', dataIndex: 'outcome', width: 170, render: (_, row) => <Tag color={outcomeColor[row.outcome]}>{row.outcome}</Tag> },
    { title: 'Alt', width: 110, render: (_, row) => formatEuroCents(row.oldPriceCents) },
    { title: 'Ziel', width: 110, render: (_, row) => formatEuroCents(row.targetPriceCents) },
    { title: 'Erwartete Marge', width: 170, render: (_, row) => `${formatEuroCents(row.expectedMarginCents)} · ${(row.expectedMarginBasisPoints / 100).toFixed(2)} %` },
    { title: 'Regel', width: 90, render: (_, row) => `v${row.ruleVersion}` },
    { title: 'Begründung', dataIndex: 'reason', ellipsis: true },
    { title: 'Aktion', width: 120, fixed: 'right', render: (_, row) => row.outcome === 'PROPOSED' ? <Button disabled={!writable} onClick={() => setApplyDecision(row)}>Anwenden</Button> : '—' },
  ], [writable]);

  return <TmPageContainer title={t('mindbay.monitoring.title')} subTitle={t('mindbay.monitoring.subTitle')}>
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert type="warning" showIcon message={t('mindbay.monitoring.dryRunTitle')} description={t('mindbay.monitoring.dryRunBody')} />
      {!writable ? <Alert type="info" showIcon message={t('mindbay.monitoring.readonlyTitle')} description={t('mindbay.monitoring.readonlyBody')} /> : null}
      {error ? <ErrorAlert title={t('mindbay.monitoring.errorTitle')} actionHint={error} /> : null}
      <SectionCard title="Steuerung" description="Regeln sind unveränderliche Versionen; ein Lauf bindet Snapshot und Regelversion." headerExtra={<Space wrap><Button disabled={!writable} onClick={() => setRuleOpen(true)}>Preisregel anlegen</Button><Button type="primary" disabled={!writable || !listings.length || !rules.length} onClick={() => setRunOpen(true)}>Monitoring ausführen</Button></Space>}>
        <Space wrap>
          <Select aria-label="eBay Listing" value={selectedListing} onChange={setSelectedListing} style={{ minWidth: 300 }} placeholder="eBay Listing auswählen" options={listings.map(row => ({ value: row.id, label: `${row.sku} · ${row.externalListingId} · ${formatEuroCents(row.priceCents)}` }))} />
          <Select aria-label="Preisregel" value={selectedRule} onChange={setSelectedRule} style={{ minWidth: 260 }} placeholder="Preisregel auswählen" options={rules.map(row => ({ value: row.id, label: `${row.name} · v${row.version}${row.autoApply ? ' · Auto' : ' · Vorschlag'}` }))} />
        </Space>
      </SectionCard>
      <TmProTable rowKey="id" headerTitle="Preisentscheidungen" loading={loading} search={false} options={false} pagination={false} dataSource={decisions} columns={columns} scroll={{ x: 1200 }} locale={{ emptyText: 'Noch keine Preisentscheidung vorhanden.' }} />
    </Space>
    <Modal title="Monitoring-Lauf starten?" open={runOpen} onCancel={() => setRunOpen(false)} onOk={() => void run()} okText="Lauf starten" confirmLoading={action === 'run'} okButtonProps={{ disabled: !selectedListing || !selectedRule }}>
      <Alert type="info" showIcon message="Der Lauf speichert immutable Snapshots und eine Entscheidung." description="Auto-Apply wird nur außerhalb von DRY_RUN und nur bei erfüllten Guardrails ausgeführt." />
    </Modal>
    <Modal title="Preisentscheidung anwenden?" open={!!applyDecision} onCancel={() => setApplyDecision(undefined)} onOk={() => void apply()} okText="Preis anwenden" confirmLoading={action === 'apply'} okButtonProps={{ danger: true }}>
      <Alert type="warning" showIcon message={`${formatEuroCents(applyDecision?.oldPriceCents)} → ${formatEuroCents(applyDecision?.targetPriceCents)}`} description="In LIVE aktualisiert diese Aktion das eBay-Angebot. DRY_RUN protokolliert nur den geplanten Request." />
    </Modal>
    <Modal title="Neue Preisregel-Version" open={ruleOpen} onCancel={() => setRuleOpen(false)} onOk={() => void createRule()} okText="Regel speichern" confirmLoading={action === 'rule'}>
      <Form form={ruleForm} layout="vertical" initialValues={{ minMarginPercent: 15, targetMarginPercent: 20, maxDeltaCents: 1000, maxDeltaPercent: 20, cooldownMinutes: 360, platformFeePercent: 12, shippingCents: 0, reserveCents: 100, autoApply: true }}>
        <Form.Item name="name" label="Regelname" rules={[{ required: true }]}><Input /></Form.Item>
        <Space wrap align="start"><Form.Item name="minMarginPercent" label="Min. Marge %" rules={[{ required: true }]}><InputNumber min={0} max={99} /></Form.Item><Form.Item name="targetMarginPercent" label="Zielmarge %" rules={[{ required: true }]}><InputNumber min={0} max={99} /></Form.Item><Form.Item name="platformFeePercent" label="eBay Fee %" rules={[{ required: true }]}><InputNumber min={0} max={99} /></Form.Item></Space>
        <Space wrap align="start"><Form.Item name="maxPriceCents" label="Max. Preis (Cent)"><InputNumber min={1} /></Form.Item><Form.Item name="maxDeltaCents" label="Max. Delta (Cent)" rules={[{ required: true }]}><InputNumber min={0} /></Form.Item><Form.Item name="maxDeltaPercent" label="Max. Delta %" rules={[{ required: true }]}><InputNumber min={0} /></Form.Item></Space>
        <Space wrap align="start"><Form.Item name="shippingCents" label="Versand (Cent)" rules={[{ required: true }]}><InputNumber min={0} /></Form.Item><Form.Item name="reserveCents" label="Rücklage (Cent)" rules={[{ required: true }]}><InputNumber min={0} /></Form.Item><Form.Item name="cooldownMinutes" label="Cooldown (Min.)" rules={[{ required: true }]}><InputNumber min={0} /></Form.Item></Space>
        <Form.Item name="autoApply" label="Auto-Apply" valuePropName="checked"><Switch /></Form.Item>
      </Form>
    </Modal>
  </TmPageContainer>;
}

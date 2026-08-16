import { useCallback, useEffect, useMemo, useState } from 'react';
import type { ProColumns } from '@ant-design/pro-components';
import { Alert, Button, DatePicker, Form, Input, InputNumber, Modal, Select, Space, Statistic, Tag, message } from 'antd';
import dayjs from 'dayjs';
import { ErrorAlert, TmPageContainer, TmProTable } from '@/components/ui';
import { usePermission } from '@/hooks/usePermission';
import { PERMISSIONS } from '@/utils/permission';
import { queryShops, type ShopListRow } from '@/services/shops';
import {
  applyCalendar,
  formatEuroCents,
  listCalendarSlots,
  previewCalendar,
  type CalendarPreviewSlot,
  type CalendarSlot,
  type PublishConfig,
} from '@/services/mindbay';

type PlannerForm = PublishConfig & { shopId: string; startAt: dayjs.Dayjs; days: number; maxPerDay: number; minSpacingMinutes: number };

const statusColor: Record<CalendarSlot['status'], string> = { DRAFT: 'default', SCHEDULED: 'blue', HELD: 'gold', PUBLISHING: 'processing', PUBLISHED: 'green', FAILED: 'red', CANCELLED: 'default' };

export default function PlannerPage() {
  const { readonly, can } = usePermission();
  const writable = !readonly && can(PERMISSIONS.PRODUCT_WRITE);
  const [form] = Form.useForm<PlannerForm>();
  const [shops, setShops] = useState<ShopListRow[]>([]);
  const [preview, setPreview] = useState<CalendarPreviewSlot[]>([]);
  const [scheduled, setScheduled] = useState<CalendarSlot[]>([]);
  const [unplaced, setUnplaced] = useState(0);
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState('');
  const [error, setError] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [shopResult, slotResult] = await Promise.all([
        queryShops({ platform: 'ebay', status: 'active', page: 1, pageSize: 100 }),
        listCalendarSlots(),
      ]);
      setShops(shopResult.list || []);
      setScheduled(slotResult.items || []);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const runPreview = async () => {
    const values = await form.validateFields(['startAt', 'days', 'maxPerDay', 'minSpacingMinutes']);
    setAction('preview');
    setError('');
    try {
      const result = await previewCalendar({ startAt: values.startAt.toISOString(), days: values.days, maxPerDay: values.maxPerDay, minSpacingMinutes: values.minSpacingMinutes });
      setPreview(result.slots || []);
      setUnplaced(result.unplaced || 0);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setAction('');
    }
  };

  const apply = async () => {
    const values = await form.validateFields();
    setAction('apply');
    setError('');
    try {
      await applyCalendar({ shopId: values.shopId, marketplace: 'EBAY_DE', slots: preview, publishConfig: { merchantLocationKey: values.merchantLocationKey, paymentPolicyId: values.paymentPolicyId, returnPolicyId: values.returnPolicyId, fulfillmentPolicyId: values.fulfillmentPolicyId, condition: values.condition, quantity: values.quantity, productSafetyStatementIds: values.productSafetyStatementIds?.trim() || undefined } });
      message.success(`${preview.length} Slot(s) idempotent eingeplant.`);
      setConfirmOpen(false);
      setPreview([]);
      await load();
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setAction('');
    }
  };

  const previewColumns = useMemo<ProColumns<CalendarPreviewSlot>[]>(() => [
    { title: 'Zeitpunkt', dataIndex: 'scheduledFor', width: 180, render: (_, row) => dayjs(row.scheduledFor).format('DD.MM.YYYY HH:mm') },
    { title: 'Listing', dataIndex: 'title', ellipsis: true },
    { title: 'Kategorie', dataIndex: 'category', width: 180 },
    { title: 'Score', dataIndex: 'score', width: 90, render: (_, row) => <Tag color="blue">{row.score}</Tag> },
    { title: 'Begründung', dataIndex: 'reason', width: 260 },
  ], []);
  const scheduledColumns = useMemo<ProColumns<CalendarSlot>[]>(() => [
    { title: 'Zeitpunkt', dataIndex: 'scheduledFor', width: 180, render: (_, row) => dayjs(row.scheduledFor).format('DD.MM.YYYY HH:mm') },
    { title: 'Status', dataIndex: 'status', width: 130, render: (_, row) => <Tag color={statusColor[row.status]}>{row.status}</Tag> },
    { title: 'Listing Draft', dataIndex: 'listingDraftId', copyable: true, ellipsis: true },
    { title: 'Preis', width: 120, render: (_, row) => formatEuroCents(row.priceCents) },
    { title: 'eBay ID', dataIndex: 'externalListingId', width: 180, copyable: true, ellipsis: true, render: (_, row) => row.externalUrl ? <a href={row.externalUrl} target="_blank" rel="noreferrer">{row.externalListingId}</a> : row.externalListingId || '—' },
    { title: 'Fehler', dataIndex: 'errorMessage', width: 260, ellipsis: true, render: (_, row) => row.errorMessage || '—' },
    { title: 'Planer-Score', dataIndex: 'plannerScore', width: 120 },
  ], []);

  return <TmPageContainer title="MindBay Planner" subTitle="READY-Listings vorschauen, Slots reservieren und kontrolliert an den eBay-Sandbox-Workflow übergeben.">
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Alert type="warning" showIcon message="Sicherer Standard: DRY_RUN + eBay Sandbox" description="Preview verändert keine Daten. Erst Einplanen reserviert Slots; DRY_RUN führt keine mutierende eBay-Anfrage aus." />
      {!writable ? <Alert type="info" showIcon message="Nur-Lese-Modus" description="Preview und Kalender bleiben sichtbar; Einplanen ist deaktiviert." /> : null}
      {error ? <ErrorAlert title="Planner-Aktion fehlgeschlagen" actionHint={error} /> : null}
      <Form form={form} layout="vertical" initialValues={{ startAt: dayjs().add(1, 'hour').startOf('hour'), days: 7, maxPerDay: 4, minSpacingMinutes: 120, marketplace: 'EBAY_DE', condition: 'NEW', quantity: 1 }}>
        <Space wrap align="start">
          <Form.Item name="shopId" label="eBay Shop" rules={[{ required: true, message: 'eBay Shop auswählen.' }]}><Select loading={loading} style={{ width: 230 }} placeholder="eBay Shop" options={shops.map(shop => ({ value: shop.id, label: `${shop.shopName} · ${shop.authStatus}` }))} /></Form.Item>
          <Form.Item name="startAt" label="Start" rules={[{ required: true }]}><DatePicker showTime minuteStep={15} format="DD.MM.YYYY HH:mm" /></Form.Item>
          <Form.Item name="days" label="Tage" rules={[{ required: true }]}><InputNumber min={1} max={31} /></Form.Item>
          <Form.Item name="maxPerDay" label="Max/Tag" rules={[{ required: true }]}><InputNumber min={1} max={24} /></Form.Item>
          <Form.Item name="minSpacingMinutes" label="Abstand (Min.)" rules={[{ required: true }]}><InputNumber min={15} max={1440} step={15} /></Form.Item>
        </Space>
        <Space wrap align="start">
          <Form.Item name="merchantLocationKey" label="Location Key" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="paymentPolicyId" label="Payment Policy" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="returnPolicyId" label="Return Policy" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="fulfillmentPolicyId" label="Fulfillment Policy" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="condition" label="Zustand" rules={[{ required: true }]}><Select style={{ width: 170 }} options={[{ value: 'NEW', label: 'Neu' }, { value: 'USED_EXCELLENT', label: 'Gebraucht: sehr gut' }, { value: 'USED_GOOD', label: 'Gebraucht: gut' }]} /></Form.Item>
          <Form.Item name="quantity" label="Menge" rules={[{ required: true }]}><InputNumber min={1} max={99} /></Form.Item>
          <Form.Item name="productSafetyStatementIds" label="eBay Safety IDs" tooltip="Optionale, komma-separierte Metadata-IDs wie EBPSS102"><Input placeholder="EBPSS102" /></Form.Item>
        </Space>
        <Space><Button loading={action === 'preview'} onClick={() => void runPreview()}>Preview berechnen</Button><Button type="primary" disabled={!writable || preview.length === 0} onClick={() => setConfirmOpen(true)}>Slots einplanen</Button></Space>
      </Form>

      <Space><Statistic title="Preview-Slots" value={preview.length} /><Statistic title="Nicht platziert" value={unplaced} /></Space>
      <TmProTable rowKey={(row) => `${row.listingDraftId}:${row.scheduledFor}`} search={false} options={false} pagination={false} columns={previewColumns} dataSource={preview} locale={{ emptyText: 'Noch keine Preview berechnet.' }} scroll={{ x: 950 }} />
      <TmProTable rowKey="id" headerTitle="Reservierte Slots" search={false} options={false} pagination={false} loading={loading} columns={scheduledColumns} dataSource={scheduled} locale={{ emptyText: 'Noch keine Slots reserviert.' }} scroll={{ x: 1250 }} />
    </Space>

    <Modal title="Slots verbindlich einplanen?" open={confirmOpen} onCancel={() => setConfirmOpen(false)} onOk={() => void apply()} okText="Einplanen" cancelText="Abbrechen" confirmLoading={action === 'apply'}>
      <Alert type="warning" showIcon message={`${preview.length} Listing(s) wechseln von READY nach SCHEDULED.`} description="Der gleiche Idempotency-Key kann keine doppelten Slots erzeugen. Bei DRY_RUN bleibt eBay unverändert." />
    </Modal>
  </TmPageContainer>;
}

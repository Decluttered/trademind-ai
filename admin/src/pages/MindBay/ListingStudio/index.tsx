import { useCallback, useEffect, useState } from 'react';
import type { ProColumns } from '@ant-design/pro-components';
import { Alert, Button, Descriptions, Form, Input, List, Modal, Popconfirm, Select, Space, Tag, message } from 'antd';
import { ErrorAlert, TmPageContainer, TmProTable } from '@/components/ui';
import {
  createGPSRProfile,
  createListingDraft,
  formatEuroCents,
  generateListing,
  getListingDraft,
  listGPSRProfiles,
  syncEbayCategoryAspects,
  listListingDrafts,
  parseEuroInput,
  validateListing,
  type ContentVersion,
  type DraftState,
  type GPSRProfile,
  type EbayCategoryAspect,
  type ListingDraft,
} from '@/services/mindbay';
import { useLocale } from '@/locale';

const states: DraftState[] = ['DRAFTING', 'NEEDS_REVIEW', 'READY', 'BLOCKED'];

export default function ListingStudioPage() {
  const { t } = useLocale();
  const [items, setItems] = useState<ListingDraft[]>([]);
  const [profiles, setProfiles] = useState<GPSRProfile[]>([]);
  const [versions, setVersions] = useState<ContentVersion[]>([]);
  const [categoryAspects, setCategoryAspects] = useState<EbayCategoryAspect[]>([]);
  const [aspectsLoading, setAspectsLoading] = useState(false);
  const [selectedDraft, setSelectedDraft] = useState<ListingDraft>();
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState('');
  const [error, setError] = useState('');
  const [state, setState] = useState<DraftState>();
  const [draftOpen, setDraftOpen] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [draftForm] = Form.useForm();
  const [profileForm] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [drafts, gpsr] = await Promise.all([listListingDrafts({ state, limit: 50 }), listGPSRProfiles()]);
      setItems(drafts.items || []);
      setProfiles(gpsr.items || []);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setLoading(false);
    }
  }, [state]);
  useEffect(() => { void load(); }, [load]);

  const run = async (id: string, kind: 'generate' | 'validate') => {
    setAction(`${kind}:${id}`);
    setError('');
    try {
      if (kind === 'generate') {
        await generateListing(id);
        message.success(t('mindbay.listing.contentGenerated'));
      } else {
        const out = await validateListing(id, {});
        out.errors.length ? message.warning(out.errors.join(' · ')) : message.success(`Validierung abgeschlossen: ${out.state}`);
      }
      await load();
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setAction('');
    }
  };

  const showDetail = async (row: ListingDraft) => {
    setAction(`detail:${row.id}`);
    try {
      const detail = await getListingDraft(row.id);
      setSelectedDraft(detail.draft);
      setVersions(detail.versions || []);
      setDetailOpen(true);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setAction('');
    }
  };

  const columns: ProColumns<ListingDraft>[] = [
    { title: t('mindbay.listing.colStatus'), dataIndex: 'state', width: 140, render: (_, row) => <Tag color={row.state === 'READY' ? 'green' : row.state === 'BLOCKED' ? 'red' : 'blue'}>{row.state}</Tag> },
    { title: t('mindbay.listing.colSource'), dataIndex: 'sourceProductId', copyable: true, ellipsis: true },
    { title: t('mindbay.listing.colCategory'), dataIndex: 'category', width: 160, render: (_, row) => row.category || '—' },
    { title: t('mindbay.listing.colPrice'), width: 120, render: (_, row) => formatEuroCents(row.priceCents) },
    { title: t('mindbay.listing.colGpsr'), width: 220, render: (_, row) => row.validationErrors?.length ? <Tag color="red">{t('mindbay.listing.blockers', { values: { count: row.validationErrors.length } })}</Tag> : <Tag>{t('mindbay.listing.stillCheck')}</Tag> },
    { title: t('mindbay.listing.colActions'), valueType: 'option', width: 330, render: (_, row) => [
      <Button key="detail" type="link" loading={action === `detail:${row.id}`} onClick={() => void showDetail(row)}>{t('mindbay.listing.versions')}</Button>,
      <Button key="gen" type="link" disabled={row.state === 'READY'} loading={action === `generate:${row.id}`} onClick={() => void run(row.id, 'generate')}>{t('mindbay.listing.generateVersion')}</Button>,
      <Popconfirm key="val" title={t('mindbay.listing.validateConfirm')} onConfirm={() => run(row.id, 'validate')}><Button type="link" loading={action === `validate:${row.id}`}>{t('mindbay.listing.validate')}</Button></Popconfirm>,
    ] },
  ];

  const createDraft = async () => {
    const values = await draftForm.validateFields();
    const cents = parseEuroInput(values.price);
    if (cents === null) {
      draftForm.setFields([{ name: 'price', errors: [t('mindbay.listing.priceInvalid')] }]);
      return;
    }
    setAction('create');
    try {
      await createListingDraft({
        sourceProductId: values.sourceProductId,
        category: values.category,
        priceCents: cents,
        requiredSpecifics: categoryAspects.filter((aspect) => aspect.required).map((aspect) => aspect.name),
        specifics: values.specifics || {},
        imageAssetIds: [],
        gpsrProfileId: values.gpsrProfileId,
      });
      message.success(t('mindbay.listing.draftCreated'));
      setDraftOpen(false);
      draftForm.resetFields();
      await load();
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setAction('');
    }
  };

  const loadCategoryAspects = async () => {
    const category = String(draftForm.getFieldValue('category') || '').trim();
    if (!category) {
      draftForm.setFields([{ name: 'category', errors: [t('mindbay.listing.categoryRequired')] }]);
      return;
    }
    setAspectsLoading(true);
    try {
      const result = await syncEbayCategoryAspects(category);
      setCategoryAspects(result.items || []);
      message.success(`${result.synced || 0} eBay-Merkmale geladen.`);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setAspectsLoading(false);
    }
  };

  const createProfile = async () => {
    const values = await profileForm.validateFields();
    setAction('profile');
    try {
      await createGPSRProfile({ ...values, documentReferences: values.documentReferences.split(/[,\n]/).map((value: string) => value.trim()).filter(Boolean) });
      message.success('GPSR-Profil angelegt.');
      setProfileOpen(false);
      profileForm.resetFields();
      await load();
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setAction('');
    }
  };

  return <TmPageContainer title={t('mindbay.listing.title')} subTitle={t('mindbay.listing.subTitle')} extra={<Space><Button onClick={() => setProfileOpen(true)}>{t('mindbay.listing.gpsrProfile')}</Button><Button type="primary" onClick={() => setDraftOpen(true)}>{t('mindbay.listing.createDraft')}</Button></Space>}>
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {error ? <ErrorAlert title={t('mindbay.listing.errorTitle')} actionHint={error} /> : null}
      <Alert type="info" showIcon message={t('mindbay.listing.readyHint')} />
      <Select allowClear value={state} onChange={setState} placeholder={t('mindbay.listing.allStates')} options={states.map(value => ({ value, label: value }))} style={{ width: 220 }} />
      <TmProTable rowKey="id" search={false} options={false} pagination={false} loading={loading} columns={columns} dataSource={items} scroll={{ x: 1050 }} />
    </Space>

    <Modal title={t('mindbay.listing.draftModalTitle')} open={draftOpen} onCancel={() => setDraftOpen(false)} onOk={() => void createDraft()} confirmLoading={action === 'create'} okText={t('mindbay.listing.okCreate')} cancelText={t('mindbay.listing.cancel')} destroyOnHidden>
      <Form form={draftForm} layout="vertical" style={{ marginTop: 16 }}>
        <Form.Item name="sourceProductId" label={t('mindbay.listing.sourceProductId')} rules={[{ required: true, message: t('mindbay.listing.sourceRequired') }]}><Input /></Form.Item>
        <Form.Item name="category" label={t('mindbay.listing.category')} rules={[{ required: true, message: t('mindbay.listing.categoryRequired') }]}>
          <Input.Search loading={aspectsLoading} enterButton={t('mindbay.listing.loadAspects')} onChange={() => setCategoryAspects([])} onSearch={() => void loadCategoryAspects()} />
        </Form.Item>
        {categoryAspects.map((aspect) => (
          <Form.Item
            key={aspect.attrId || aspect.name}
            name={['specifics', aspect.name]}
            label={`${aspect.name}${aspect.required ? ' (Pflicht)' : ''}`}
            rules={aspect.required ? [{ required: true, message: `${aspect.name} ist für diese eBay-Kategorie erforderlich.` }] : undefined}
          >
            {aspect.options?.length ? <Select showSearch options={aspect.options.map((value) => ({ value, label: value }))} /> : <Input />}
          </Form.Item>
        ))}
        <Form.Item name="price" label={t('mindbay.listing.priceEur')} rules={[{ required: true, message: t('mindbay.listing.priceRequired') }]}><Input inputMode="decimal" placeholder="29,99" /></Form.Item>
        <Form.Item name="gpsrProfileId" label={t('mindbay.listing.gpsrLabel')}><Select allowClear options={profiles.map(profile => ({ value: profile.id, label: profile.name }))} /></Form.Item>
      </Form>
    </Modal>

    <Modal title="GPSR-Profil anlegen" open={profileOpen} onCancel={() => setProfileOpen(false)} onOk={() => void createProfile()} confirmLoading={action === 'profile'} okText="Profil speichern" destroyOnHidden>
      <Form form={profileForm} layout="vertical">
        <Form.Item name="name" label="Profilname" rules={[{ required: true }]}><Input /></Form.Item>
        <Form.Item name="manufacturerName" label="Hersteller" rules={[{ required: true }]}><Input /></Form.Item>
        <Form.Item name="manufacturerAddress" label="Herstelleranschrift" rules={[{ required: true }]}><Input.TextArea /></Form.Item>
        <Form.Item name="responsiblePersonName" label="Verantwortliche Person" rules={[{ required: true }]}><Input /></Form.Item>
        <Form.Item name="responsiblePersonAddress" label="Anschrift der verantwortlichen Person" rules={[{ required: true }]}><Input.TextArea /></Form.Item>
        <Form.Item name="safetyInformation" label="Warn- und Sicherheitstexte" rules={[{ required: true }]}><Input.TextArea /></Form.Item>
        <Form.Item name="documentReferences" label="Dokumentenreferenzen" rules={[{ required: true }]}><Input.TextArea placeholder="Eine URL oder Referenz pro Zeile" /></Form.Item>
      </Form>
    </Modal>

    <Modal title="Inhaltsversionen" open={detailOpen} onCancel={() => setDetailOpen(false)} footer={null} width={760}>
      {selectedDraft ? <Descriptions size="small" column={2} items={[{ key: 'state', label: 'Status', children: selectedDraft.state }, { key: 'price', label: 'Preis', children: formatEuroCents(selectedDraft.priceCents) }]} /> : null}
      <List dataSource={versions} locale={{ emptyText: 'Noch keine Inhaltsversion' }} renderItem={version => <List.Item key={version.id}><List.Item.Meta title={`Version ${version.version}: ${version.title}`} description={<Space direction="vertical"><span>{version.description}</span><Tag>{version.generator}</Tag></Space>} /></List.Item>} />
    </Modal>
  </TmPageContainer>;
}

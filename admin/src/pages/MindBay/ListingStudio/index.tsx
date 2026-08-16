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

const states: DraftState[] = ['DRAFTING', 'NEEDS_REVIEW', 'READY', 'BLOCKED'];

export default function ListingStudioPage() {
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
        message.success('Neue Inhaltsversion erzeugt.');
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
    { title: 'Status', dataIndex: 'state', width: 140, render: (_, row) => <Tag color={row.state === 'READY' ? 'green' : row.state === 'BLOCKED' ? 'red' : 'blue'}>{row.state}</Tag> },
    { title: 'Source Product', dataIndex: 'sourceProductId', copyable: true, ellipsis: true },
    { title: 'Kategorie', dataIndex: 'category', width: 160, render: (_, row) => row.category || '—' },
    { title: 'Preis', width: 120, render: (_, row) => formatEuroCents(row.priceCents) },
    { title: 'GPSR / Prüfung', width: 220, render: (_, row) => row.validationErrors?.length ? <Tag color="red">{row.validationErrors.length} Blocker</Tag> : <Tag>noch prüfen</Tag> },
    { title: 'Aktionen', valueType: 'option', width: 330, render: (_, row) => [
      <Button key="detail" type="link" loading={action === `detail:${row.id}`} onClick={() => void showDetail(row)}>Versionen</Button>,
      <Button key="gen" type="link" disabled={row.state === 'READY'} loading={action === `generate:${row.id}`} onClick={() => void run(row.id, 'generate')}>AI-Version erzeugen</Button>,
      <Popconfirm key="val" title="Listing jetzt validieren?" onConfirm={() => run(row.id, 'validate')}><Button type="link" loading={action === `validate:${row.id}`}>Validieren</Button></Popconfirm>,
    ] },
  ];

  const createDraft = async () => {
    const values = await draftForm.validateFields();
    const cents = parseEuroInput(values.price);
    if (cents === null) {
      draftForm.setFields([{ name: 'price', errors: ['Bitte einen gültigen EUR-Betrag eingeben.'] }]);
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
      message.success('Listing-Entwurf angelegt.');
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
      draftForm.setFields([{ name: 'category', errors: ['Kategorie ist erforderlich.'] }]);
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

  return <TmPageContainer title="MindBay Listing Studio" subTitle="Lokale Entwürfe erzeugen und prüfen; Phase 1 veröffentlicht nichts bei eBay." extra={<Space><Button onClick={() => setProfileOpen(true)}>GPSR-Profil</Button><Button type="primary" onClick={() => setDraftOpen(true)}>Entwurf anlegen</Button></Space>}>
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      {error ? <ErrorAlert title="Listing-Aktion fehlgeschlagen" actionHint={error} /> : null}
      <Alert type="info" showIcon message="READY erfordert Kategorie, positiven Cent-Preis, Bilder, Pflicht-Specifics, faktentreuen Inhalt und ein vollständiges GPSR-Profil." />
      <Select allowClear value={state} onChange={setState} placeholder="Alle Status" options={states.map(value => ({ value, label: value }))} style={{ width: 220 }} />
      <TmProTable rowKey="id" search={false} options={false} pagination={false} loading={loading} columns={columns} dataSource={items} scroll={{ x: 1050 }} />
    </Space>

    <Modal title="Listing-Entwurf anlegen" open={draftOpen} onCancel={() => setDraftOpen(false)} onOk={() => void createDraft()} confirmLoading={action === 'create'} okText="Anlegen" cancelText="Abbrechen" destroyOnHidden>
      <Form form={draftForm} layout="vertical" style={{ marginTop: 16 }}>
        <Form.Item name="sourceProductId" label="Source Product ID" rules={[{ required: true, message: 'Source Product ID ist erforderlich.' }]}><Input /></Form.Item>
        <Form.Item name="category" label="eBay-Kategorie-ID" rules={[{ required: true, message: 'Kategorie ist erforderlich.' }]}>
          <Input.Search loading={aspectsLoading} enterButton="Pflichtfelder laden" onChange={() => setCategoryAspects([])} onSearch={() => void loadCategoryAspects()} />
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
        <Form.Item name="price" label="Preis in EUR" rules={[{ required: true, message: 'Preis ist erforderlich.' }]}><Input inputMode="decimal" placeholder="29,99" /></Form.Item>
        <Form.Item name="gpsrProfileId" label="GPSR-Profil"><Select allowClear options={profiles.map(profile => ({ value: profile.id, label: profile.name }))} /></Form.Item>
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

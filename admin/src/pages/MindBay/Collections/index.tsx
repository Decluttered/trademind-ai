import { useCallback, useEffect, useState } from 'react';
import { Button, Form, Input, List, Select, Space, Tag, Typography, message } from 'antd';
import { EmptyState, ErrorAlert, TmPageContainer } from '@/components/ui';
import {
  addProductToCollection,
  createCollection,
  listCollections,
  listMindBayProducts,
  type Collection,
  type CollectionKind,
  type ProductListItem,
} from '@/services/mindbay';

type CreateValues = { name: string; description?: string; kind: CollectionKind };
type AssignValues = { collectionId: string; sourceProductId: string; reason?: string };

export default function MindBayCollectionsPage() {
  const [items, setItems] = useState<Collection[]>([]);
  const [products, setProducts] = useState<ProductListItem[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [createForm] = Form.useForm<CreateValues>();
  const [assignForm] = Form.useForm<AssignValues>();

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [collections, productPage] = await Promise.all([listCollections({ limit: 50 }), listMindBayProducts({ limit: 100 })]);
      setItems(collections.items || []);
      setProducts(productPage.items || []);
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  const create = async (values: CreateValues) => {
    setSaving(true);
    try {
      await createCollection(values);
      createForm.resetFields();
      message.success('Collection angelegt');
      await load();
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const assign = async (values: AssignValues) => {
    setSaving(true);
    try {
      await addProductToCollection(values.collectionId, { sourceProductId: values.sourceProductId, reason: values.reason });
      assignForm.resetFields(['sourceProductId', 'reason']);
      message.success('Produkt zur Collection hinzugefügt');
    } catch (cause) {
      setError((cause as Error).message);
    } finally {
      setSaving(false);
    }
  };

  return <TmPageContainer title="MindBay Collections" subTitle="Produkte in manuelle oder regelbasierte Arbeitsmengen einordnen.">
    <Space direction="vertical" size={20} style={{ width: '100%' }}>
      <Form form={createForm} layout="inline" initialValues={{ kind: 'manual' }} onFinish={create}>
        <Form.Item name="name" rules={[{ required: true, message: 'Name erforderlich' }]}><Input placeholder="Collection-Name" /></Form.Item>
        <Form.Item name="kind" rules={[{ required: true }]}><Select style={{ width: 150 }} options={['manual','search','seller','rule','import'].map(value => ({ value, label: value }))} /></Form.Item>
        <Form.Item name="description"><Input placeholder="Beschreibung" /></Form.Item>
        <Button htmlType="submit" type="primary" loading={saving}>Anlegen</Button>
      </Form>
      <Form form={assignForm} layout="inline" onFinish={assign}>
        <Form.Item name="collectionId" rules={[{ required: true, message: 'Collection erforderlich' }]}><Select placeholder="Collection" style={{ width: 220 }} options={items.map(item => ({ value: item.id, label: item.name }))} /></Form.Item>
        <Form.Item name="sourceProductId" rules={[{ required: true, message: 'Produkt erforderlich' }]}><Select showSearch optionFilterProp="label" placeholder="Produkt" style={{ width: 300 }} options={products.map(product => ({ value: product.id, label: `${product.externalId} · ${product.currentSnapshot.title}` }))} /></Form.Item>
        <Form.Item name="reason"><Input placeholder="Aufnahmegrund" /></Form.Item>
        <Button htmlType="submit" loading={saving}>Produkt hinzufügen</Button>
      </Form>
      {error ? <ErrorAlert title="Collections konnten nicht verarbeitet werden" actionHint={error} /> : null}
      {!loading && !items.length ? <EmptyState title="Noch keine Collections" description="Legen Sie die erste Arbeitsmenge oben an." /> : <List loading={loading} dataSource={items} renderItem={item => <List.Item key={item.id}><List.Item.Meta title={<Space><span>{item.name}</span><Tag>{item.kind}</Tag></Space>} description={item.description || <Typography.Text type="secondary">Keine Beschreibung</Typography.Text>} /></List.Item>} />}
    </Space>
  </TmPageContainer>;
}

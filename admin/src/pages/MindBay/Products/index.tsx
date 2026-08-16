import { useCallback, useEffect, useState } from 'react';
import type { ProColumns } from '@ant-design/pro-components';
import { Button, Input, InputNumber, Select, Space, Tag } from 'antd';
import { EmptyState, ErrorAlert, TmPageContainer, TmProTable } from '@/components/ui';
import { formatEuroCents, listCollections, listMindBayProducts, type Collection, type ProductListItem } from '@/services/mindbay';

export default function MindBayProductsPage() {
  const [items, setItems] = useState<ProductListItem[]>([]);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [q, setQ] = useState('');
  const [collectionId, setCollectionId] = useState<string>();
  const [minScore, setMinScore] = useState<number>();
  const [cursor, setCursor] = useState<string>();
  const [next, setNext] = useState<string>();

  useEffect(() => { listCollections({ limit: 100 }).then(page => setCollections(page.items || [])).catch(() => setCollections([])); }, []);
  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const page = await listMindBayProducts({ q: q || undefined, collectionId, minScore, cursor, limit: 25 });
      setItems(page.items || []);
      setNext(page.nextCursor);
    } catch (cause) {
      setItems([]);
      setError((cause as Error).message);
    } finally {
      setLoading(false);
    }
  }, [collectionId, cursor, minScore, q]);
  useEffect(() => { void load(); }, [load]);

  const columns: ProColumns<ProductListItem>[] = [
    { title: 'ASIN', dataIndex: 'externalId', copyable: true, width: 140 },
    { title: 'Produkt', dataIndex: ['currentSnapshot', 'title'], ellipsis: true },
    { title: 'Preis', search: false, width: 120, render: (_, row) => formatEuroCents(row.currentSnapshot.priceCents) },
    { title: 'Verfügbarkeit', search: false, width: 150, render: (_, row) => row.currentSnapshot.availability || '—' },
    { title: 'Score', search: false, width: 110, render: (_, row) => row.assessment ? <Tag color={row.assessment.score >= 60 ? 'green' : 'gold'}>{row.assessment.score} · {row.assessment.confidence}%</Tag> : '—' },
    { title: 'Quelle', search: false, width: 110, render: (_, row) => <a href={row.canonicalUrl} target="_blank" rel="noreferrer">amazon.de</a> },
  ];
  const applyFilters = () => { setCursor(undefined); void load(); };

  return <TmPageContainer title="MindBay Produkte" subTitle="Deduplizierte ASINs mit aktuellem Snapshot und erklärbarem Opportunity Score.">
    <Space direction="vertical" size={16} style={{ width: '100%' }}>
      <Space wrap>
        <Input value={q} allowClear onChange={event => setQ(event.target.value)} onPressEnter={applyFilters} placeholder="Nach ASIN suchen" style={{ width: 240 }} />
        <Select allowClear value={collectionId} onChange={setCollectionId} placeholder="Collection" style={{ width: 220 }} options={collections.map(item => ({ value: item.id, label: item.name }))} />
        <InputNumber min={0} max={100} value={minScore} onChange={value => setMinScore(value ?? undefined)} placeholder="Mindestscore" />
        <Button onClick={applyFilters}>Filtern</Button>
      </Space>
      {error ? <ErrorAlert title="Produkte konnten nicht geladen werden" actionHint={error} /> : null}
      {!loading && !error && !items.length ? <EmptyState title="Noch keine Produkte" description="Erfassen Sie zuerst eine Amazon.de-Produktseite oder lockern Sie die Filter." actionLabel="Discovery öffnen" actionPath="/mindbay/discovery" /> : <TmProTable rowKey="id" search={false} options={false} pagination={false} loading={loading} columns={columns} dataSource={items} scroll={{ x: 900 }} />}
      {next ? <Button onClick={() => setCursor(next)}>Nächste Seite</Button> : null}
    </Space>
  </TmPageContainer>;
}

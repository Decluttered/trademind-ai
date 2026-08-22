import { type ActionType, type ProColumns, type ProFormInstance } from '@ant-design/pro-components';
import { TmPageContainer, TmProTable as ProTable } from '@/components/ui';
import InventorySyncDisabledBanner from '@/components/inventory/InventorySyncDisabledBanner';
import {
  INVENTORY_BIND_STATUS,
  INVENTORY_SKU_AMBIGUOUS_MESSAGE,
  INVENTORY_SKU_NOT_BOUND_MESSAGE,
  INVENTORY_STOCK_STATUS,
  INVENTORY_SYNC_STATUS,
  inventoryTagFromMap,
} from '@/constants/inventoryLabels';
import { INVENTORY_COPY, PRODUCT_COPY } from '@/constants/copywriting';
import { useListEmptyLocale } from '@/hooks/useListEmptyLocale';
import { queryInventoryCenter, type InventoryCenterRow } from '@/services/inventory';
import { Space, Tag, Typography, message } from 'antd';
import { formatDateTime } from '@/utils/formatTime';
import { Link } from '@umijs/max';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useUrlQueryState } from '@/hooks/useUrlState';
import { useKeywordSearchField } from '@/hooks/useKeywordSearchField';
import KeywordSafetyHint from '@/components/common/KeywordSafetyHint';
import { parsePositiveInt } from '@/utils/urlState';

const INVENTORY_QUERY_KEYS = [
  'page',
  'pageSize',
  'keyword',
  'stockStatus',
  'syncStatus',
  'skuBindStatus',
  'platform',
  'shopId',
  'productSkuId',
  'source',
  'skuId',
] as const;

function tagFrom(raw: string, map: Record<string, { text: string; color: string }>) {
  const cfg = inventoryTagFromMap(raw, map);
  return <Tag color={cfg.color}>{cfg.text}</Tag>;
}

export default function InventoryCenterPage() {
  const emptyLocale = useListEmptyLocale('inventoryCenter', { permissionScoped: true });
  const actionRef = useRef<ActionType>();
  const formRef = useRef<ProFormInstance>();
  const { state: urlState, setState: setUrlState, clearState: clearUrlState } =
    useUrlQueryState<Record<(typeof INVENTORY_QUERY_KEYS)[number], string | undefined>>(
      INVENTORY_QUERY_KEYS,
    );
  const [tablePage, setTablePage] = useState(1);
  const [tablePageSize, setTablePageSize] = useState(20);
  const {
    fieldProps: keywordFieldProps,
    prepareKeyword,
    showSensitiveHint,
  } = useKeywordSearchField({
    setUrlState,
    formRef,
    actionRef,
    setTablePage,
  });

  const skuIdFromUrl = useMemo(() => {
    return urlState.productSkuId || urlState.skuId;
  }, [urlState.productSkuId, urlState.skuId]);

  useEffect(() => {
    setTablePage(parsePositiveInt(urlState.page, 1));
    setTablePageSize(parsePositiveInt(urlState.pageSize, 20));
    formRef.current?.setFieldsValue?.({
      keyword: urlState.keyword,
      stockStatus: urlState.stockStatus,
      syncStatus: urlState.syncStatus,
      skuBindStatus: urlState.skuBindStatus,
      platform: urlState.platform,
      shopId: urlState.shopId,
      productSkuId: skuIdFromUrl,
    });
  }, [
    skuIdFromUrl,
    urlState.keyword,
    urlState.page,
    urlState.pageSize,
    urlState.platform,
    urlState.shopId,
    urlState.skuBindStatus,
    urlState.stockStatus,
    urlState.syncStatus,
  ]);

  useEffect(() => {
    if (!skuIdFromUrl) return;
    actionRef.current?.reload?.();
  }, [skuIdFromUrl]);

  const columns: ProColumns<InventoryCenterRow>[] = useMemo(
    () => [
      {
        title: 'Keyword',
        dataIndex: 'keyword',
        hideInTable: true,
        fieldProps: { placeholder: 'Product title / SKU code / name', ...keywordFieldProps },
      },
      { title: 'SKU ID', dataIndex: 'productSkuId', hideInTable: true },
      { title: 'Shop ID', dataIndex: 'shopId', hideInTable: true },
      { title: 'Platform', dataIndex: 'platform', hideInTable: true },
      {
        title: 'Inventory status',
        dataIndex: 'stockStatus',
        hideInTable: true,
        valueType: 'select',
        valueEnum: Object.fromEntries(Object.entries(INVENTORY_STOCK_STATUS).map(([k, v]) => [k, { text: v.text }])),
      },
      {
        title: INVENTORY_COPY.skuBinding,
        dataIndex: 'skuBindStatus',
        hideInTable: true,
        valueType: 'select',
        valueEnum: Object.fromEntries(Object.entries(INVENTORY_BIND_STATUS).map(([k, v]) => [k, { text: v.text }])),
      },
      {
        title: 'Sync status',
        dataIndex: 'syncStatus',
        hideInTable: true,
        valueType: 'select',
        valueEnum: Object.fromEntries(Object.entries(INVENTORY_SYNC_STATUS).map(([k, v]) => [k, { text: v.text }])),
      },
      {
        title: 'Exceptions only',
        dataIndex: 'hasException',
        hideInTable: true,
        valueType: 'select',
        valueEnum: { true: { text: 'Yes' }, false: { text: 'No' } },
      },
      {
        title: 'Product',
        dataIndex: 'productTitle',
        width: 180,
        search: false,
        ellipsis: true,
        render: (_, r) => (
          <Link to={`/product/drafts/${r.productId}?tab=inventory`}>{r.productTitle || '—'}</Link>
        ),
      },
      {
        title: PRODUCT_COPY.sku,
        dataIndex: 'skuCode',
        width: 120,
        search: false,
        ellipsis: true,
        render: (_, r) => r.skuCode || '—',
      },
      {
        title: 'SKU',
        dataIndex: 'skuName',
        width: 120,
        search: false,
        ellipsis: true,
        render: (_, r) => r.skuName || '—',
      },
      { title: 'Local stock', dataIndex: 'stock', width: 88, search: false },
      { title: 'Available stock', dataIndex: 'availableStock', width: 88, search: false },
      { title: 'Alert threshold', dataIndex: 'warningStock', width: 88, search: false },
      {
        title: '库存状态',
        dataIndex: 'stockStatus',
        width: 100,
        search: false,
        render: (_, r) => tagFrom(r.stockStatus, INVENTORY_STOCK_STATUS),
      },
      {
        title: INVENTORY_COPY.skuBinding,
        dataIndex: 'skuBindStatus',
        width: 96,
        search: false,
        render: (_, r) => tagFrom(r.skuBindStatus, INVENTORY_BIND_STATUS),
      },
      {
        title: 'Platform sync',
        dataIndex: 'platformSyncStatus',
        width: 96,
        search: false,
        render: (_, r) => tagFrom(r.platformSyncStatus, INVENTORY_SYNC_STATUS),
      },
      {
        title: 'Latest deduction',
        dataIndex: 'lastDeductAt',
        width: 156,
        search: false,
        render: (_, r) => (r.lastDeductAt ? formatDateTime(r.lastDeductAt) : '—'),
      },
      {
        title: 'Latest sync',
        dataIndex: 'lastSyncAt',
        width: 156,
        search: false,
        render: (_, r) => (r.lastSyncAt ? formatDateTime(r.lastSyncAt) : '—'),
      },
      {
        title: 'Exceptions',
        dataIndex: 'exceptionCount',
        width: 72,
        search: false,
        render: (_, r) =>
          r.exceptionCount > 0 ? <Tag color="red">{r.exceptionCount}</Tag> : <Tag>0</Tag>,
      },
      {
        title: 'Actions',
        valueType: 'option',
        width: 280,
        fixed: 'right',
        render: (_, r) => (
          <Space wrap size="small">
            <Link to={`/product/drafts/${r.productId}?tab=inventory`}>View product</Link>
            <Link to={`/inventory/deductions?productSkuId=${encodeURIComponent(r.productSkuId)}`}>
              Deduction records
            </Link>
            <Link to={`/inventory/sync-tasks?productSkuId=${encodeURIComponent(r.productSkuId)}`}>
              Sync tasks
            </Link>
            {r.exceptionCount > 0 ? (
              <Link to={`/ops/task-center/failures?taskType=inventory_sync`}>Failed tasks</Link>
            ) : null}
          </Space>
        ),
      },
    ],
    [keywordFieldProps],
  );

  return (
    <TmPageContainer
      title="Inventory center"
      subTitle="View local inventory, SKU bindings, and platform-sync status. It does not automatically sync or replenish stock."
    >
      <InventorySyncDisabledBanner />
      <KeywordSafetyHint visible={showSensitiveHint} />
      <Typography.Paragraph type="secondary">
        {INVENTORY_SKU_NOT_BOUND_MESSAGE}{' '}
        {INVENTORY_SKU_AMBIGUOUS_MESSAGE}
      </Typography.Paragraph>
      <ProTable<InventoryCenterRow>
        rowKey="productSkuId"
        actionRef={actionRef}
        formRef={formRef}
        columns={columns}
        scroll={{ x: 1500 }}
        search={{ labelWidth: 100, defaultCollapsed: false }}
        onReset={() => {
          setTablePage(1);
          setTablePageSize(20);
          clearUrlState(INVENTORY_QUERY_KEYS, { replace: true });
        }}
        pagination={{
          current: tablePage,
          pageSize: tablePageSize,
          showSizeChanger: true,
          onChange: (page, pageSize) => {
            setTablePage(page);
            setTablePageSize(pageSize);
            setUrlState({
              page: page > 1 ? page : undefined,
              pageSize: pageSize !== 20 ? pageSize : undefined,
            });
          },
        }}
        locale={emptyLocale}
        request={async (params) => {
          try {
            const qp = {
              keyword: prepareKeyword(params.keyword),
              productSkuId:
                (params.productSkuId as string | undefined)?.trim() || skuIdFromUrl,
              shopId: (params.shopId as string | undefined)?.trim(),
              platform: (params.platform as string | undefined)?.trim(),
              stockStatus: (params.stockStatus as string | undefined)?.trim(),
              skuBindStatus: (params.skuBindStatus as string | undefined)?.trim(),
              syncStatus: (params.syncStatus as string | undefined)?.trim(),
              page: params.current ?? tablePage,
              pageSize: params.pageSize ?? tablePageSize,
            };
            setUrlState(
              {
                page: Number(qp.page) > 1 ? qp.page : undefined,
                pageSize: Number(qp.pageSize) !== 20 ? qp.pageSize : undefined,
                keyword: qp.keyword,
                productSkuId: qp.productSkuId,
                shopId: qp.shopId,
                platform: qp.platform,
                stockStatus: qp.stockStatus,
                skuBindStatus: qp.skuBindStatus,
                syncStatus: qp.syncStatus,
                source: urlState.source,
              },
              { replace: true },
            );
            const res = await queryInventoryCenter({
              keyword: qp.keyword,
              productSkuId: qp.productSkuId,
              shopId: qp.shopId,
              platform: qp.platform,
              stockStatus: qp.stockStatus,
              skuBindStatus: qp.skuBindStatus,
              syncStatus: qp.syncStatus,
              hasException: params.hasException === 'true' || params.hasException === true,
              page: qp.page,
              pageSize: qp.pageSize,
            });
            return { data: res.list ?? [], success: true, total: res.pagination?.total ?? 0 };
          } catch (e: unknown) {
            message.error((e as Error)?.message || 'Loading failed.');
            return { data: [], success: false, total: 0 };
          }
        }}
      />
    </TmPageContainer>
  );
}

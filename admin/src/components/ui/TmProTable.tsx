import { ColumnHeightOutlined, ReloadOutlined, SettingOutlined } from '@ant-design/icons';
import { ProTable } from '@ant-design/pro-components';
import type { ActionType, ProTableProps } from '@ant-design/pro-components';
import { Button, Tooltip } from 'antd';
import { useCallback, useMemo, useRef, useState } from 'react';

export type TmProTableProps<T extends Record<string, unknown>, U extends Record<string, unknown> = Record<string, unknown>> =
  ProTableProps<T, U>;

type TmToolBarRender<T extends Record<string, unknown>, U extends Record<string, unknown>> = Exclude<
  ProTableProps<T, U>['toolBarRender'],
  false | undefined
>;

/**
 * Unified ProTable: uses a clickable Button to handle refresh (fixes the toolbar's built-in span icon being unclickable in some layouts).
 */
export default function TmProTable<
  T extends Record<string, unknown>,
  U extends Record<string, unknown> = Record<string, unknown>,
>({
  actionRef: userActionRef,
  options,
  toolBarRender,
  onLoadingChange,
  className,
  ...rest
}: TmProTableProps<T, U>) {
  const innerRef = useRef<ActionType>();
  const actionRef = userActionRef ?? innerRef;
  const [loading, setLoading] = useState(false);

  const mergedOptions = useMemo(() => {
    if (options === false) {
      return false;
    }
    const base = options ?? {};
    const setting = base.setting === false
      ? false
      : {
          ...(typeof base.setting === 'object' ? base.setting : {}),
          settingIcon:
            (typeof base.setting === 'object' ? base.setting.settingIcon : undefined) ?? (
              <Button type="text" aria-label="列设置" icon={<SettingOutlined />} />
            ),
        };
    return {
      density: true,
      ...base,
      densityIcon: base.densityIcon ?? (
        <Button type="text" aria-label="表格密度" icon={<ColumnHeightOutlined />} />
      ),
      setting,
      // The built-in reload is a span + icon whose click area is easily broken; triggered instead by the Button in toolBarRender.
      reload: false,
    };
  }, [options]);

  const mergedToolBarRender = useCallback(
    (action: ActionType | undefined, config: Parameters<TmToolBarRender<T, U>>[1]) => {
      const userNodes = typeof toolBarRender === 'function' ? toolBarRender(action, config) ?? [] : [];
      if (mergedOptions === false) {
        return userNodes;
      }
      return [
        ...userNodes,
        <Tooltip key="tm-reload" title="刷新">
          <Button
            type="text"
            aria-label="刷新"
            icon={<ReloadOutlined spin={loading} />}
            onClick={() => {
              void action?.reload?.();
            }}
          />
        </Tooltip>,
      ];
    },
    [toolBarRender, loading, mergedOptions],
  );

  return (
    <ProTable<T, U>
      {...rest}
      className={['tm-pro-table', className].filter(Boolean).join(' ')}
      actionRef={actionRef}
      options={mergedOptions}
      toolBarRender={mergedToolBarRender}
      onLoadingChange={(isLoading) => {
        setLoading(!!isLoading);
        onLoadingChange?.(isLoading);
      }}
    />
  );
}

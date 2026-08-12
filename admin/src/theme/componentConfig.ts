import type { ConfigProviderProps } from 'antd';

export const ADMIN_DRAWER_WIDTH = 'var(--tm-app-drawer-width)';

/** ConfigProvider wrapper styles are applied after each Drawer's width prop. */
export function createAdminDrawerConfig(
  drawerConfig?: ConfigProviderProps['drawer'],
): NonNullable<ConfigProviderProps['drawer']> {
  return {
    ...drawerConfig,
    styles: {
      ...drawerConfig?.styles,
      wrapper: {
        ...drawerConfig?.styles?.wrapper,
        width: ADMIN_DRAWER_WIDTH,
      },
    },
  };
}

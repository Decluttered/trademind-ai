import { Drawer, type DrawerProps } from 'antd';
import { ADMIN_DRAWER_WIDTH } from '@/theme';

const DEFAULT_DRAWER_WIDTH = ADMIN_DRAWER_WIDTH;

/** Project-wide drawer; width is controlled by the responsive Admin drawer token. */
export default function AppDrawer({ width = DEFAULT_DRAWER_WIDTH, rootClassName, ...rest }: DrawerProps) {
  return (
    <Drawer
      width={width}
      rootClassName={['tm-app-drawer', rootClassName].filter(Boolean).join(' ')}
      {...rest}
    />
  );
}

export { DEFAULT_DRAWER_WIDTH };

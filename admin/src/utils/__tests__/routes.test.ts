import { describe, expect, it } from 'vitest';
import routes from '../../../config/routes';

type RouteNode = {
  path?: string;
  name?: string;
  hideInMenu?: boolean;
  routes?: RouteNode[];
};

function collectVisibleMenuPaths(nodes: RouteNode[], paths: string[] = []): string[] {
  for (const node of nodes) {
    if (node.name && node.path && !node.hideInMenu) {
      paths.push(node.path);
    }
    if (node.routes) {
      collectVisibleMenuPaths(node.routes, paths);
    }
  }
  return paths;
}

describe('Admin route menu configuration', () => {
  it('uses a unique path for every visible menu item', () => {
    const paths = collectVisibleMenuPaths(routes);
    const duplicates = paths.filter((path, index) => paths.indexOf(path) !== index);

    expect([...new Set(duplicates)]).toEqual([]);
  });

  it('keeps the legacy inventory URL outside the menu and exposes a unique overview path', () => {
    const inventory = routes.find((route) => route.path === '/inventory');

    expect(inventory?.routes).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ path: '/inventory', component: './Inventory' }),
        expect.objectContaining({
          path: '/inventory/overview',
          name: '库存中心',
          component: './Inventory',
        }),
      ]),
    );
    expect(inventory?.routes?.find((route) => route.path === '/inventory')?.name).toBeUndefined();
  });
});

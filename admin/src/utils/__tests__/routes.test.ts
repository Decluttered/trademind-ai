import { describe, expect, it } from 'vitest';
import routes, {
  createDevelopmentOpsRoutes,
  createInternalInventorySyncRoutes,
} from '../../../config/routes';

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

function collectVisibleMenuNames(nodes: RouteNode[], names: string[] = []): string[] {
  for (const node of nodes) {
    if (node.name && !node.hideInMenu) {
      names.push(node.name);
    }
    if (node.routes) {
      collectVisibleMenuNames(node.routes, names);
    }
  }
  return names;
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

  it('keeps internal phase and fixture routes out of the production menu', () => {
    const paths = collectVisibleMenuPaths(routes);
    const names = collectVisibleMenuNames(routes);

    expect(paths.filter((path) => path.startsWith('/ops/inventory-sync'))).toEqual([]);
    expect(names.filter((name) => /P\d+|Batch|Gate|Fixture|夹具|人工验收/i.test(name))).toEqual([]);
  });

  it('excludes internal inventory fixture routes from production builds', () => {
    expect(createInternalInventorySyncRoutes(false)).toEqual([]);
    expect(createInternalInventorySyncRoutes(true)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ path: '/ops/inventory-sync', hideInMenu: true }),
      ]),
    );
    expect(createInternalInventorySyncRoutes(true).every((route) => route.hideInMenu)).toBe(true);
  });

  it('excludes incomplete operational tools from production builds', () => {
    expect(createDevelopmentOpsRoutes(false)).toEqual([]);
    expect(createDevelopmentOpsRoutes(true)).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ path: '/ops/backups', name: '备份管理' }),
        expect.objectContaining({ path: '/ops/restores', name: '恢复验证' }),
        expect.objectContaining({ path: '/ops/releases', name: '发布流程记录' }),
        expect.objectContaining({ path: '/ops/disaster-recovery', name: '灾备演练记录' }),
      ]),
    );
  });

  it('keeps historical AI batches outside the menu for deep-link compatibility', () => {
    const aiRoutes = routes.find((route) => route.path === '/ai')?.routes;

    expect(aiRoutes).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          path: '/ai/batches',
          name: '历史 AI 批次',
          hideInMenu: true,
        }),
      ]),
    );
  });
});

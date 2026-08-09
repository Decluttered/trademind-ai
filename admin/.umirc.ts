import { defineConfig } from '@umijs/max';
import { createStaticLightThemeConfig } from './src/theme/themeConfig';
import routes from './config/routes';

const lightTheme = createStaticLightThemeConfig();

const appRoutes = routes.map((route) =>
  route.path === '/ops'
    ? {
        ...route,
        routes: [
          ...(route.routes || []),
          {
            path: '/ops/p10-readiness',
            name: 'P10 人工验收',
            icon: 'SafetyCertificateOutlined',
            component: './Ops/P10Readiness',
          },
        ],
      }
    : route,
);

export default defineConfig({
  title: '贸灵 TradeMind',
  npmClient: 'npm',
  antd: {
    appConfig: {},
    configProvider: {
      theme: lightTheme,
    },
  },
  access: {},
  model: {},
  initialState: {},
  request: {},
  layout: {
    /** 侧栏/顶栏品牌仅在 `app.tsx` 的 `logo` 中渲染，此处不设 title，避免与 logo 内文案重复 */
    title: false,
    locale: false,
    layout: 'mix',
    navTheme: 'light',
    fixedHeader: true,
    fixSiderbar: true,
    contentWidth: 'Fluid',
  },
  routes: appRoutes,
  devtool: process.env.NODE_ENV === 'production' ? false : 'source-map',
  proxy: {
    '/api': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
    },
    '/static': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
    },
  },
});

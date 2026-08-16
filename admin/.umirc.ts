import { defineConfig } from '@umijs/max';
import { createStaticLightThemeConfig } from './src/theme/themeConfig';
import routes from './config/routes';

const lightTheme = createStaticLightThemeConfig();

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
    /** Sidebar/topnav branding is only rendered via `logo` in `app.tsx`; no title is set here to avoid duplicating the logo's text */
    title: false,
    locale: false,
    layout: 'mix',
    navTheme: 'light',
    fixedHeader: true,
    fixSiderbar: true,
    contentWidth: 'Fluid',
  },
  routes,
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

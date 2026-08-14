import { BrowserManager } from './browser/manager.js';
import { loadCollectorDevelopmentEnv } from './config/local-env.js';
import { listenCollectorHttp } from './http/server.js';

loadCollectorDevelopmentEnv();

const browser = new BrowserManager();
const server = listenCollectorHttp(browser);

function shutdown() {
  console.info('[collector] shutting down...');
  server.close(() => {
    browser.close().finally(() => process.exit(0));
  });
}

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);

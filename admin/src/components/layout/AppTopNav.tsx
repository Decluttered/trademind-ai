import { useEffect, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import { flushSync } from 'react-dom';
import {
  DownOutlined,
  LogoutOutlined,
  MoonOutlined,
  SunOutlined,
} from '@ant-design/icons';
import { useAntdConfigSetter } from '@umijs/max';
import { Avatar, Dropdown, Tooltip } from 'antd';
import { themeTokens, tmSemanticTokens } from '@/constants/layoutTokens';
import {
  applyThemeMode,
  createAdminThemeConfig,
  getStoredThemeMode,
  persistThemeMode,
} from '@/theme';
import './AppTopNav.less';

const avatarStyle: CSSProperties = {
  color: '#fff',
  background: `linear-gradient(135deg, ${themeTokens.colorPrimary} 0%, ${tmSemanticTokens.dataAccent} 100%)`,
};

const THEME_SWITCHING_CLASS = 'tm-theme-switching';

function requestThemeFrame(callback: FrameRequestCallback) {
  if (typeof window.requestAnimationFrame === 'function') {
    return window.requestAnimationFrame(callback);
  }
  return window.setTimeout(() => callback(performance.now()), 16);
}

function cancelThemeFrame(frameId: number) {
  if (typeof window.cancelAnimationFrame === 'function') {
    window.cancelAnimationFrame(frameId);
    return;
  }
  window.clearTimeout(frameId);
}

export function resolveUserLabels(user?: API.CurrentUser) {
  const displayName = user?.displayName?.trim() || '管理员';
  const email = user?.email?.trim() || '';
  const username = user?.username?.trim() || '';
  const loginId = email || username;

  if (displayName.includes('@') && loginId && displayName === loginId) {
    const local = displayName.split('@')[0]?.trim() || displayName;
    return {
      primary: local,
      secondary: displayName,
      initial: local.slice(0, 1).toUpperCase(),
    };
  }

  return {
    primary: displayName,
    secondary: loginId && loginId !== displayName ? loginId : '',
    initial: displayName.slice(0, 1).toUpperCase(),
  };
}

type AppTopNavProps = {
  user?: API.CurrentUser;
  onLogout: () => void | Promise<void>;
};

export default function AppTopNav({ user, onLogout }: AppTopNavProps) {
  const setAntdConfig = useAntdConfigSetter();
  const [themeMode, setThemeMode] = useState(getStoredThemeMode);
  const [isMenuOpen, setIsMenuOpen] = useState(false);
  const themeFrameRef = useRef<number>();

  useEffect(() => {
    applyThemeMode(themeMode);
  }, [themeMode]);

  useEffect(
    () => () => {
      if (themeFrameRef.current !== undefined) {
        cancelThemeFrame(themeFrameRef.current);
      }
      document.documentElement.classList.remove(THEME_SWITCHING_CLASS);
    },
    [],
  );

  if (!user) {
    return <nav className="tm-app-top-nav" aria-label="内容导航栏" />;
  }

  const { primary, secondary, initial } = resolveUserLabels(user);
  const nextThemeMode = themeMode === 'light' ? 'dark' : 'light';
  const nextThemeLabel = nextThemeMode === 'dark' ? '深色模式' : '浅色模式';

  const switchTheme = () => {
    const root = document.documentElement;
    if (themeFrameRef.current !== undefined) {
      cancelThemeFrame(themeFrameRef.current);
    }
    root.classList.add(THEME_SWITCHING_CLASS);

    flushSync(() => {
      setThemeMode(nextThemeMode);
      setAntdConfig({ theme: createAdminThemeConfig(nextThemeMode) });
    });
    persistThemeMode(nextThemeMode);

    themeFrameRef.current = requestThemeFrame(() => {
      themeFrameRef.current = requestThemeFrame(() => {
        root.classList.remove(THEME_SWITCHING_CLASS);
        themeFrameRef.current = undefined;
      });
    });
  };

  return (
    <nav className="tm-app-top-nav" aria-label="内容导航栏">
      <Tooltip title={`切换到${nextThemeLabel}`} placement="bottom">
        <button
          type="button"
          className="tm-app-top-nav__theme-toggle"
          aria-label={`切换到${nextThemeLabel}`}
          aria-pressed={themeMode === 'dark'}
          onClick={switchTheme}
        >
          {themeMode === 'light' ? (
            <MoonOutlined aria-hidden="true" />
          ) : (
            <SunOutlined aria-hidden="true" />
          )}
        </button>
      </Tooltip>
      <Dropdown
        menu={{
          items: [
            {
              key: 'logout',
              icon: <LogoutOutlined className="tm-app-account-dropdown__logout-icon" />,
              label: (
                <span className="tm-app-account-dropdown__label">
                  <span className="tm-app-account-dropdown__title">退出登录</span>
                  <span className="tm-app-account-dropdown__description">返回登录页面</span>
                </span>
              ),
              onClick: () => {
                setIsMenuOpen(false);
                void onLogout();
              },
            },
          ],
        }}
        open={isMenuOpen}
        onOpenChange={setIsMenuOpen}
        overlayClassName="tm-app-account-dropdown"
        placement="bottomRight"
        trigger={['click']}
        overlayStyle={{ minWidth: 196 }}
      >
        <button
          type="button"
          className="tm-app-top-nav__user"
          aria-label={`当前用户 ${primary}`}
          aria-haspopup="menu"
          aria-expanded={isMenuOpen}
        >
          <Avatar
            size={32}
            className="tm-app-top-nav__avatar"
            style={avatarStyle}
          >
            {initial}
          </Avatar>
          <span className="tm-app-top-nav__user-meta">
            <span className="tm-app-top-nav__user-name" title={primary}>
              {primary}
            </span>
            <span className="tm-app-top-nav__user-account" title={secondary || '管理员'}>
              {secondary || '管理员'}
            </span>
          </span>
          <DownOutlined
            className={`tm-app-top-nav__user-chevron${isMenuOpen ? ' is-open' : ''}`}
            aria-hidden="true"
          />
        </button>
      </Dropdown>
    </nav>
  );
}

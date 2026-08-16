import type {
  CSSProperties,
  ReactElement,
  ReactNode,
} from "react";
import { useCallback } from "react";
import type { MenuDataItem } from "@umijs/route-utils";
import { history } from "@umijs/max";
import { MenuFoldOutlined, MenuUnfoldOutlined } from "@ant-design/icons";
import type { ConfigProviderProps } from "antd";
import type { RequestConfig, RunTimeLayoutConfig } from "@/typings/umi-runtime";
import AppGlobalSearch from "@/components/layout/AppGlobalSearch";
import AppTopNav from "@/components/layout/AppTopNav";
import AppModalBridge from "@/components/AppModalBridge";
import AppMessageBridge from "@/components/AppMessageBridge";
import BrandLogo from "@/components/BrandLogo";
import { AUTH_TOKEN_KEY } from "@/constants/auth";
import { layoutTokens } from "@/constants/layoutTokens";
import { postJSON } from "@/services/request";
import {
  applyThemeMode,
  createAdminDrawerConfig,
  createAdminThemeConfig,
  getStoredThemeMode,
} from "@/theme";
import {
  LocaleProvider,
  MENU_LOCALE_KEYS,
  antdLocaleFor,
  applyAdminLocale,
  getStoredAdminLocale,
  translate,
} from "@/locale";
import { canAccessPath, filterMenuByPermission } from "@/utils/menuAccess";
import { isPublicAdminPath } from "@/utils/publicRoutes";
import { useInitialStateModel } from "@/hooks/useInitialStateModel";
import type { InitialStateModel } from "@/typings/umi-runtime";

/** ProLayout 会为侧栏品牌回调提供 props，移动端顶栏品牌回调则不会。 */
type SiderMenuLayoutProps = {
  collapsed?: boolean;
};

type HeaderLayoutProps = SiderMenuLayoutProps & {
  isMobile?: boolean;
  menuData?: MenuDataItem[];
  onCollapse?: (collapsed: boolean) => void;
};

type HeaderContentProps = HeaderLayoutProps & {
  role?: string | null;
  permissions?: string[];
};

async function loadProfileFromToken(
  token: string,
): Promise<API.CurrentUser | undefined> {
  const res = await fetch("/api/v1/auth/profile", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const json = (await res.json()) as { code: number; data?: API.CurrentUser };
  if (!res.ok || json.code !== 0 || !json.data) return undefined;
  return json.data;
}

/**
 * Runs inside umi antd innerProvider `<App>` (under ConfigProvider).
 * Do not add another `<App>` in rootContainer — that wraps outside ConfigProvider and breaks static message.
 */
export function innerProvider(container: ReactElement) {
  return (
    <LocaleProvider>
      <AppModalBridge />
      <AppMessageBridge />
      {container}
    </LocaleProvider>
  );
}

function localizeMenuData(menuData: MenuDataItem[]): MenuDataItem[] {
  const locale = getStoredAdminLocale();
  return menuData.map((item) => {
    const path = String(item.path || "");
    const key = MENU_LOCALE_KEYS[path];
    const localizedName = key ? translate(locale, key) : item.name;
    return {
      ...item,
      name: localizedName,
      children: item.children ? localizeMenuData(item.children) : item.children,
      routes: item.routes
        ? localizeMenuData(item.routes as MenuDataItem[])
        : item.routes,
    };
  });
}

export const antd = (memo: ConfigProviderProps): ConfigProviderProps => {
  const mode = getStoredThemeMode();
  const locale = getStoredAdminLocale();
  applyThemeMode(mode);
  applyAdminLocale(locale);
  return {
    ...memo,
    locale: antdLocaleFor(locale),
    drawer: createAdminDrawerConfig(memo.drawer),
    theme: createAdminThemeConfig(mode),
  };
};

export async function getInitialState(): Promise<{
  currentUser?: API.CurrentUser;
}> {
  const token = localStorage.getItem(AUTH_TOKEN_KEY);
  if (!token) {
    return {};
  }
  const user = await loadProfileFromToken(token);
  if (!user) {
    localStorage.removeItem(AUTH_TOKEN_KEY);
    return {};
  }
  return { currentUser: user };
}

export const request: RequestConfig = {
  requestInterceptors: [
    (url, options) => {
      const token = localStorage.getItem(AUTH_TOKEN_KEY);
      const headers: Record<string, string> = {
        ...((options.headers as Record<string, string>) || {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      };
      return { url, options: { ...options, headers } };
    },
  ],
  errorConfig: {
    errorHandler: (error: any) => {
      if (error?.info?.skipErrorHandler) {
        throw error;
      }
      const status = error?.response?.status;
      const reqUrl = String(error?.config?.url || "");
      if (status === 401 && !reqUrl.includes("/auth/login")) {
        localStorage.removeItem(AUTH_TOKEN_KEY);
        const path = history.location.pathname;
        if (!isPublicAdminPath(path)) {
          const q = encodeURIComponent(path);
          window.location.assign(
            `${window.location.origin}/user/login?redirect=${q}`,
          );
          return;
        }
      }
      throw error;
    },
  },
};

/** 顶栏品牌图形（与登录页同一 `logo.png`） */
const TM_BRAND_MARK = <BrandLogo height={28} className="tm-app-brand-logo" />;

const TM_APP_LAYOUT_STYLE = {
  "--tm-app-header-height": `${layoutTokens.appHeaderHeight}px`,
} as CSSProperties;

async function logoutAndClear(
  setInitialState: InitialStateModel["setInitialState"],
) {
  try {
    await postJSON("/api/v1/auth/logout");
  } catch {
    /* ignore */
  }
  localStorage.removeItem(AUTH_TOKEN_KEY);
  setInitialState((s) => ({ ...s, currentUser: undefined }));
  history.push("/user/login");
}

function AppTopNavBridge() {
  const { setInitialState, initialState } = useInitialStateModel();
  return (
    <AppTopNav
      user={initialState?.currentUser}
      onLogout={() => logoutAndClear(setInitialState)}
    />
  );
}

function AppBrandButton() {
  return (
    <button
      type="button"
      className="tm-app-brand-header"
      aria-label="返回工作台"
      onClick={() => history.push("/dashboard")}
    >
      <BrandLogo height={28} className="tm-app-brand-logo" />
      <span className="tm-app-brand-header__name">
        <span className="tm-app-brand-header__name-primary">贸灵</span>
        <span className="tm-app-brand-header__name-secondary">TradeMind</span>
      </span>
    </button>
  );
}

function AppHeaderIdentity({
  collapsed = false,
  onCollapse,
}: HeaderLayoutProps) {
  const collapseLabel = collapsed ? "展开侧栏" : "收起侧栏";

  return (
    <div className="tm-app-header-identity">
      <AppBrandButton />
      <button
        type="button"
        className="tm-app-header-identity__sider-toggle"
        aria-label={collapseLabel}
        title={collapseLabel}
        onClick={() => onCollapse?.(!collapsed)}
      >
        {collapsed ? (
          <MenuUnfoldOutlined aria-hidden="true" />
        ) : (
          <MenuFoldOutlined aria-hidden="true" />
        )}
      </button>
    </div>
  );
}

function AppHeaderContent({
  role,
  permissions,
  ...layoutProps
}: HeaderContentProps) {
  const canSearchPath = useCallback(
    (path: string) => canAccessPath(path, role, permissions),
    [permissions, role],
  );

  return (
    <div
      className={`tm-app-header-content${
        layoutProps.isMobile ? " tm-app-header-content--mobile" : ""
      }`}
    >
      {layoutProps.isMobile ? null : <AppHeaderIdentity {...layoutProps} />}
      <AppGlobalSearch
        items={layoutProps.menuData || []}
        compact={layoutProps.isMobile}
        canAccessPath={canSearchPath}
        onNavigate={(path) => history.push(path)}
      />
    </div>
  );
}

export const layout: RunTimeLayoutConfig = ({ initialState }) => ({
  className: "tm-app-layout",
  style: TM_APP_LAYOUT_STYLE,
  title: false,
  logo: TM_BRAND_MARK,
  layout: "mix",
  navTheme: "light",
  siderWidth: 224,
  actionsRender: () => <AppTopNavBridge />,
  avatarProps: false,
  rightContentRender: false,
  // ProLayout forces `collapsed: false` inside headerTitleRender, so interactive
  // collapse controls must stay in headerContentRender to receive the real state.
  // The empty desktop title wrapper is removed by a narrowly scoped shell style.
  headerTitleRender: false,
  headerContentRender: (props: HeaderLayoutProps) => (
    <AppHeaderContent
      {...props}
      role={initialState?.currentUser?.role}
      permissions={initialState?.currentUser?.permissions}
    />
  ),
  collapsedButtonRender: false,
  menuHeaderRender: (
    _logoDom: ReactNode,
    _titleDom: ReactNode,
    props?: SiderMenuLayoutProps,
  ) => (props ? null : <AppBrandButton />),
  token: {
    bgLayout: "var(--ant-color-bg-layout)",
    header: {
      colorBgHeader: "var(--ant-color-bg-container)",
      colorBgScrollHeader: "var(--ant-color-bg-container)",
      colorBgMenuElevated: "var(--ant-color-bg-elevated)",
      colorTextMenu: "var(--ant-color-text-secondary)",
      colorTextMenuActive: "var(--ant-color-text)",
      colorTextMenuSelected: "var(--ant-color-primary)",
      heightLayoutHeader: layoutTokens.appHeaderHeight,
    },
    sider: {
      colorMenuBackground: "var(--ant-color-bg-container)",
      colorBgMenuItemCollapsedElevated: "var(--ant-color-bg-elevated)",
      colorBgMenuItemSelected: "var(--ant-color-primary-bg)",
      colorTextMenu: "var(--ant-color-text-secondary)",
      colorTextMenuActive: "var(--ant-color-text)",
      colorTextMenuItemHover: "var(--ant-color-text)",
      colorTextMenuSelected: "var(--ant-color-primary)",
      colorTextMenuTitle: "var(--ant-color-text)",
    },
    pageContainer: {
      colorBgPageContainer: "var(--ant-color-bg-layout)",
      colorBgPageContainerFixed: "var(--ant-color-bg-container)",
    },
  },
  menu: { locale: false },
  menuDataRender: (menuData: MenuDataItem[]) =>
    localizeMenuData(
      filterMenuByPermission(
        menuData,
        initialState?.currentUser?.role,
        initialState?.currentUser?.permissions,
      ),
    ),
  onPageChange: () => {
    const { pathname } = history.location;
    if (isPublicAdminPath(pathname)) return;
    // 必须用 token 判断：initialState 在此闭包里不会在登录后刷新，会一直当作未登录并反复 push 登录页，触发 Navigate 死循环。
    if (!localStorage.getItem(AUTH_TOKEN_KEY)) {
      history.replace(`/user/login?redirect=${encodeURIComponent(pathname)}`);
    }
  },
});

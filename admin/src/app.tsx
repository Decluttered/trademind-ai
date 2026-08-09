import type {
  CSSProperties,
  KeyboardEvent,
  ReactElement,
  ReactNode,
} from "react";
import type { MenuDataItem } from "@umijs/route-utils";
import { history } from "@umijs/max";
import type { RequestConfig, RunTimeLayoutConfig } from "@/typings/umi-runtime";
import AppTopNav from "@/components/layout/AppTopNav";
import AppMessageBridge from "@/components/AppMessageBridge";
import BrandLogo from "@/components/BrandLogo";
import { AUTH_TOKEN_KEY } from "@/constants/auth";
import { layoutTokens } from "@/constants/layoutTokens";
import { postJSON } from "@/services/request";
import {
  applyThemeMode,
  createAdminThemeConfig,
  getStoredThemeMode,
} from "@/theme";
import { filterMenuByPermission } from "@/utils/menuAccess";
import { useInitialStateModel } from "@/hooks/useInitialStateModel";
import type { InitialStateModel } from "@/typings/umi-runtime";

/** ProLayout 侧栏菜单头部 / 头像区回调的常用 props */
type SiderMenuLayoutProps = {
  collapsed?: boolean;
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
    <>
      <AppMessageBridge />
      {container}
    </>
  );
}

export const antd = (memo: Record<string, unknown>) => {
  const mode = getStoredThemeMode();
  applyThemeMode(mode);
  return {
    ...memo,
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
        if (path !== "/user/login" && !path.startsWith("/user/login")) {
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

/** 侧栏 / 顶栏品牌图形（与登录页同一 `logo.png`） */
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
  // `mix` layout already renders the complete brand in the sider. Keep the
  // desktop header free of a second logo while retaining the mobile header mark.
  headerTitleRender: false,
  menuHeaderRender: (
    logoDom: ReactNode,
    _titleDom: ReactNode,
    props?: SiderMenuLayoutProps,
  ) => {
    const collapsed = props?.collapsed;
    const goHome = () => history.push("/dashboard");
    const interactive = {
      role: "button" as const,
      tabIndex: 0,
      onClick: goHome,
      onKeyDown: (e: KeyboardEvent) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          goHome();
        }
      },
    };

    if (collapsed) {
      return (
        <div
          {...interactive}
          className="tm-app-brand-header tm-app-brand-header--collapsed"
          style={{
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            height: "100%",
            cursor: "pointer",
            width: "100%",
            boxSizing: "border-box",
          }}
        >
          {logoDom}
        </div>
      );
    }

    return (
      <div
        {...interactive}
        className="tm-app-brand-header"
        style={{
          display: "flex",
          alignItems: "center",
          gap: 10,
          height: "100%",
          paddingInline: 16,
          cursor: "pointer",
          minWidth: 0,
          boxSizing: "border-box",
        }}
      >
        {logoDom}
        <span
          style={{
            fontWeight: 600,
            fontSize: 16,
            letterSpacing: "-0.02em",
            color: "var(--ant-color-text)",
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
          }}
        >
          贸灵{" "}
          <span
            style={{
              fontWeight: 500,
              color: "var(--ant-color-text-secondary)",
            }}
          >
            TradeMind
          </span>
        </span>
      </div>
    );
  },
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
    filterMenuByPermission(
      menuData,
      initialState?.currentUser?.role,
      initialState?.currentUser?.permissions,
    ),
  onPageChange: () => {
    const { pathname } = history.location;
    if (pathname === "/user/login" || pathname.startsWith("/user/login"))
      return;
    // 必须用 token 判断：initialState 在此闭包里不会在登录后刷新，会一直当作未登录并反复 push 登录页，触发 Navigate 死循环。
    if (!localStorage.getItem(AUTH_TOKEN_KEY)) {
      history.replace(`/user/login?redirect=${encodeURIComponent(pathname)}`);
    }
  },
});

import { useState } from "react";
import type { CSSProperties } from "react";
import { DownOutlined, LogoutOutlined } from "@ant-design/icons";
import { Avatar, Dropdown } from "antd";
import { themeTokens, tmSemanticTokens } from "@/constants/layoutTokens";
import ThemeToggleButton from "./ThemeToggleButton";
import "./AppTopNav.less";

const avatarStyle: CSSProperties = {
  color: "#fff",
  background: `linear-gradient(135deg, ${themeTokens.colorPrimary} 0%, ${tmSemanticTokens.dataAccent} 100%)`,
};

export function resolveUserLabels(user?: API.CurrentUser) {
  const displayName = user?.displayName?.trim() || "管理员";
  const email = user?.email?.trim() || "";
  const username = user?.username?.trim() || "";
  const loginId = email || username;

  if (displayName.includes("@") && loginId && displayName === loginId) {
    const local = displayName.split("@")[0]?.trim() || displayName;
    return {
      primary: local,
      secondary: displayName,
      initial: local.slice(0, 1).toUpperCase(),
    };
  }

  return {
    primary: displayName,
    secondary: loginId && loginId !== displayName ? loginId : "",
    initial: displayName.slice(0, 1).toUpperCase(),
  };
}

type AppTopNavProps = {
  user?: API.CurrentUser;
  onLogout: () => void | Promise<void>;
};

export default function AppTopNav({ user, onLogout }: AppTopNavProps) {
  const [isMenuOpen, setIsMenuOpen] = useState(false);

  if (!user) {
    return <nav className="tm-app-top-nav" aria-label="内容导航栏" />;
  }

  const { primary, secondary, initial } = resolveUserLabels(user);

  return (
    <nav className="tm-app-top-nav" aria-label="内容导航栏">
      <ThemeToggleButton className="tm-app-top-nav__theme-toggle" />
      <Dropdown
        menu={{
          items: [
            {
              key: "logout",
              icon: (
                <LogoutOutlined className="tm-app-account-dropdown__logout-icon" />
              ),
              label: (
                <span className="tm-app-account-dropdown__label">
                  <span className="tm-app-account-dropdown__title">
                    退出登录
                  </span>
                  <span className="tm-app-account-dropdown__description">
                    返回登录页面
                  </span>
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
        trigger={["click"]}
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
            <span
              className="tm-app-top-nav__user-account"
              title={secondary || "管理员"}
            >
              {secondary || "管理员"}
            </span>
          </span>
          <DownOutlined
            className={`tm-app-top-nav__user-chevron${isMenuOpen ? " is-open" : ""}`}
            aria-hidden="true"
          />
        </button>
      </Dropdown>
    </nav>
  );
}

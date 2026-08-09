import { theme as antdTheme } from "antd";
import type { ThemeConfig } from "antd";
import {
  elevationTokens,
  layoutTokens,
  themeTokens,
} from "../constants/layoutTokens";
import type { ThemeMode } from "./themeMode";

const darkThemeTokens = {
  colorText: "#f1f5f9",
  colorTextSecondary: "#a8b3c2",
  colorBorderSecondary: "#2b3441",
  colorBgLayout: "#0f1115",
  colorBgContainer: "#171a21",
  colorBgElevated: "#1c2028",
  colorBgTableHeader: "#1d222b",
  boxShadowTertiary: "0 1px 2px rgba(0, 0, 0, 0.32)",
} as const;

const lightThemeTokens = {
  ...themeTokens,
  colorBgElevated: themeTokens.colorBgContainer,
  colorBgTableHeader: "#f8fafc",
  boxShadowTertiary: elevationTokens.cardShadow,
} as const;

const ADMIN_CSS_VAR_KEY = "trademind-admin";

export function createAdminThemeConfig(mode: ThemeMode): ThemeConfig {
  const isDark = mode === "dark";
  const colors = isDark ? darkThemeTokens : lightThemeTokens;

  return {
    // Give each mode its own CSS-variable scope. Reusing a key lets cssinjs clean
    // the newly generated component styles while it disposes the previous theme,
    // leaving already mounted controls on stale colors until the next reload.
    cssVar: { key: `${ADMIN_CSS_VAR_KEY}-${mode}` },
    algorithm: isDark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
    token: {
      colorPrimary: isDark ? "#60a5fa" : themeTokens.colorPrimary,
      colorSuccess: isDark ? "#4ade80" : themeTokens.colorSuccess,
      colorWarning: isDark ? "#fbbf24" : themeTokens.colorWarning,
      colorError: isDark ? "#f87171" : themeTokens.colorError,
      colorInfo: isDark ? "#22d3ee" : themeTokens.colorInfo,
      colorText: colors.colorText,
      colorTextSecondary: colors.colorTextSecondary,
      colorBorderSecondary: colors.colorBorderSecondary,
      colorBgLayout: colors.colorBgLayout,
      colorBgContainer: colors.colorBgContainer,
      colorBgElevated: colors.colorBgElevated,
      borderRadius: layoutTokens.borderRadius,
      borderRadiusLG: layoutTokens.borderRadiusLg,
      controlHeight: layoutTokens.controlHeight,
      boxShadowTertiary: colors.boxShadowTertiary,
      fontFamily: themeTokens.fontFamily,
    },
    components: {
      Layout: {
        bodyBg: colors.colorBgLayout,
        headerBg: colors.colorBgContainer,
        footerBg: colors.colorBgLayout,
        siderBg: colors.colorBgContainer,
      },
      Menu: {
        itemBorderRadius: layoutTokens.borderRadius,
        itemHeight: 40,
        itemMarginBlock: 4,
        itemMarginInline: 8,
        iconSize: 16,
        collapsedIconSize: 16,
      },
      Card: {
        headerFontSize: layoutTokens.pageDescSize + 1,
        borderRadiusLG: layoutTokens.borderRadius,
      },
      Button: {
        borderRadius: layoutTokens.borderRadius,
      },
      Table: {
        headerBg: colors.colorBgTableHeader,
        cellFontSize: 14,
      },
    },
  };
}

/** Build-time config excludes algorithm functions; runtime config supplies the active algorithm. */
export function createStaticLightThemeConfig(): ThemeConfig {
  const config = createAdminThemeConfig("light");
  return {
    cssVar: config.cssVar,
    token: config.token,
    components: config.components,
  };
}

import { theme as antdTheme } from "antd";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createAdminThemeConfig } from "../themeConfig";
import {
  THEME_MODE_STORAGE_KEY,
  applyThemeMode,
  persistThemeMode,
  readThemeMode,
} from "../themeMode";

beforeEach(() => {
  delete document.documentElement.dataset.theme;
  document.documentElement.style.removeProperty("color-scheme");
});

describe("theme mode", () => {
  it("uses light mode when there is no valid stored preference", () => {
    expect(readThemeMode({ getItem: () => null })).toBe("light");
    expect(readThemeMode({ getItem: () => "unknown" })).toBe("light");
    expect(
      readThemeMode({
        getItem: () => {
          throw new Error("storage unavailable");
        },
      }),
    ).toBe("light");
  });

  it("persists dark mode and updates the document color scheme", () => {
    const setItem = vi.fn();

    persistThemeMode("dark", { setItem });

    expect(setItem).toHaveBeenCalledWith(THEME_MODE_STORAGE_KEY, "dark");
    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(document.documentElement.style.colorScheme).toBe("dark");

    applyThemeMode("light");
    expect(document.documentElement.dataset.theme).toBe("light");
    expect(document.documentElement.style.colorScheme).toBe("light");
  });
});

describe("Admin theme config", () => {
  it("keeps light as the default palette and applies the dark algorithm on demand", () => {
    const light = createAdminThemeConfig("light");
    const dark = createAdminThemeConfig("dark");
    const lightAgain = createAdminThemeConfig("light");

    expect(light.algorithm).toBe(antdTheme.defaultAlgorithm);
    expect(light.cssVar).toEqual({ key: "trademind-admin-light" });
    expect(light.token?.colorBgLayout).toBe("#f4f6f9");
    expect(light.token?.colorBgElevated).toBe("#ffffff");
    expect(dark.algorithm).toBe(antdTheme.darkAlgorithm);
    expect(dark.cssVar).toEqual({ key: "trademind-admin-dark" });
    expect(dark.token?.colorBgLayout).toBe("#0f1115");
    expect(dark.token?.colorText).toBe("#f1f5f9");
    expect(dark.token?.colorBgElevated).toBe("#1c2028");
    expect(lightAgain.token?.colorBgElevated).toBe("#ffffff");
    expect(lightAgain.cssVar).toEqual({ key: "trademind-admin-light" });
    expect(lightAgain.components?.Table?.headerBg).toBe("#f8fafc");
  });
});

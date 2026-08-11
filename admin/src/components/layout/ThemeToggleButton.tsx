import { useEffect, useRef, useState } from "react";
import { flushSync } from "react-dom";
import { MoonOutlined, SunOutlined } from "@ant-design/icons";
import { useAntdConfigSetter } from "@umijs/max";
import { Tooltip } from "antd";
import type { TooltipProps } from "antd";
import {
  applyThemeMode,
  createAdminThemeConfig,
  getStoredThemeMode,
  persistThemeMode,
} from "@/theme";

const THEME_SWITCHING_CLASS = "tm-theme-switching";

function requestThemeFrame(callback: FrameRequestCallback) {
  if (typeof window.requestAnimationFrame === "function") {
    return window.requestAnimationFrame(callback);
  }
  return window.setTimeout(() => callback(performance.now()), 16);
}

function cancelThemeFrame(frameId: number) {
  if (typeof window.cancelAnimationFrame === "function") {
    window.cancelAnimationFrame(frameId);
    return;
  }
  window.clearTimeout(frameId);
}

type ThemeToggleButtonProps = {
  className: string;
  tooltipPlacement?: TooltipProps["placement"];
};

export default function ThemeToggleButton({
  className,
  tooltipPlacement = "bottom",
}: ThemeToggleButtonProps) {
  const setAntdConfig = useAntdConfigSetter();
  const [themeMode, setThemeMode] = useState(getStoredThemeMode);
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

  const nextThemeMode = themeMode === "light" ? "dark" : "light";
  const nextThemeLabel = nextThemeMode === "dark" ? "深色模式" : "浅色模式";

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
    <Tooltip title={`切换到${nextThemeLabel}`} placement={tooltipPlacement}>
      <button
        type="button"
        className={className}
        aria-label={`切换到${nextThemeLabel}`}
        aria-pressed={themeMode === "dark"}
        onClick={switchTheme}
      >
        {themeMode === "light" ? (
          <MoonOutlined aria-hidden="true" />
        ) : (
          <SunOutlined aria-hidden="true" />
        )}
      </button>
    </Tooltip>
  );
}

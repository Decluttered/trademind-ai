/**
 * Umi/Max relies on .umi generated artifacts and the umi type package in the IDE; when `max dev` / `max setup` hasn't been run,
 * it may report "@umijs/max has no exported member xxx". This provides a minimal type supplement, consistent with the runtime.
 */
export {};

import type { ConfigProviderProps } from 'antd';
import type { Dispatch, SetStateAction } from 'react';

type UmiLocation = {
  pathname: string;
  search: string;
  hash: string;
  state?: unknown;
  key?: string;
};

type UmiHistory = {
  push(to: string | { pathname: string; search?: string; hash?: string }, state?: unknown): void;
  replace(to: string | { pathname: string; search?: string; hash?: string }, state?: unknown): void;
  go(delta: number): void;
  goBack(): void;
  goForward(): void;
  location: UmiLocation;
  listen(listener: (location: UmiLocation, action?: string) => void): () => void;
};

declare module '@umijs/max' {
  export function request<Response = unknown>(
    url: string,
    options?: {
      method?: string;
      data?: unknown;
      params?: Record<string, string | number | boolean | undefined>;
      headers?: Record<string, string>;
      skipErrorHandler?: boolean;
      [key: string]: unknown;
    },
  ): Promise<Response>;

  export const history: UmiHistory;

  export function useLocation(): UmiLocation;

  export function useParams<
    T extends Record<string, string | undefined> = Record<string, string | undefined>,
  >(): T;

  export function useSearchParams(): [
    URLSearchParams,
    (
      next: URLSearchParams | Record<string, string | string[] | undefined>,
      options?: { replace?: boolean },
    ) => void,
  ];

  export function useModel<T = unknown>(namespace: string): T;

  export function useAntdConfigSetter(): Dispatch<SetStateAction<ConfigProviderProps>>;

  export const Link: (props: {
    to: string | { pathname?: string; search?: string; hash?: string };
    children?: unknown;
    [key: string]: unknown;
  }) => JSX.Element;

  export const Outlet: (props?: Record<string, unknown>) => JSX.Element;
}

/**
 * Umi runtime config types (consistent with src/app.tsx).
 * Not imported directly from @umijs/max: the official package may be missing these export declarations when `max setup` hasn't been run.
 */

export type RequestConfig = {
  requestInterceptors?: Array<
    (
      url: string,
      options: Record<string, unknown>,
    ) =>
      | { url: string; options: Record<string, unknown> }
      | Promise<{ url: string; options: Record<string, unknown> }>
  >;
  responseInterceptors?: unknown[];
  errorConfig?: {
    errorHandler?: (error: unknown) => void;
    errorThrower?: (response: unknown) => void;
  };
  [key: string]: unknown;
};

export type RunTimeLayoutConfig = (initData: {
  initialState?: { currentUser?: API.CurrentUser };
  [key: string]: unknown;
}) => Record<string, unknown>;

/** Consistent with app.tsx `getInitialState` return value */
export type InitialState = {
  currentUser?: API.CurrentUser;
};

/** Shape of the Umi `@@initialState` model */
export type InitialStateModel = {
  initialState?: InitialState;
  setInitialState: (
    updater: InitialState | ((state: InitialState) => InitialState),
  ) => Promise<void>;
  refresh?: () => Promise<InitialState>;
};

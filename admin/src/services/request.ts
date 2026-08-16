import { request } from '@umijs/max';

/** Unified backend response structure (aligned with Gin Envelope) */
export type ApiResponse<T> = {
  code: number;
  message: string;
  data: T;
  traceId?: string;
};

export type RequestOptions = {
  headers?: Record<string, string>;
};

function withOptions(options?: RequestOptions) {
  return options?.headers ? { headers: options.headers } : {};
}

export class ApiRequestError extends Error {
  code: number;
  traceId?: string;
  data: unknown;

  constructor(res: ApiResponse<unknown>) {
    super(res.message || 'request_failed');
    this.name = 'ApiRequestError';
    this.code = res.code;
    this.traceId = res.traceId;
    this.data = res.data;
  }
}

function unwrap<T>(res: ApiResponse<T>): T {
  if (res.code !== 0) {
    throw new ApiRequestError(res as ApiResponse<unknown>);
  }
  return res.data;
}

/** Generic GET (individual modules will later be split into their own service files) */
export async function getJSON<T>(path: string): Promise<T> {
  const res = await request<ApiResponse<T>>(path, { method: 'GET' });
  return unwrap(res);
}

/** Generic PUT */
export async function putJSON<T, B extends object>(path: string, body: B, options?: RequestOptions): Promise<T> {
  const res = await request<ApiResponse<T>>(path, {
    method: 'PUT',
    data: body,
    ...withOptions(options),
  });
  return unwrap(res);
}

/** Generic PATCH */
export async function patchJSON<T, B extends object>(path: string, body: B, options?: RequestOptions): Promise<T> {
  const res = await request<ApiResponse<T>>(path, {
    method: 'PATCH',
    data: body,
    ...withOptions(options),
  });
  return unwrap(res);
}

/** Generic POST */
export async function postJSON<T>(path: string, body?: object, options?: RequestOptions): Promise<T> {
  const res = await request<ApiResponse<T>>(path, {
    method: 'POST',
    data: body,
    ...withOptions(options),
  });
  return unwrap(res);
}

/** GET with query params */
export async function getWithParams<T>(path: string, params?: Record<string, string | number | undefined>): Promise<T> {
  const res = await request<ApiResponse<T>>(path, {
    method: 'GET',
    params,
  });
  return unwrap(res);
}

/** DELETE */
export async function deleteJSON<T>(path: string): Promise<T> {
  const res = await request<ApiResponse<T>>(path, { method: 'DELETE' });
  return unwrap(res);
}

/** multipart/form-data (e.g. uploads); request detects FormData automatically, do not set Content-Type manually */
export async function postFormData<T>(path: string, data: FormData): Promise<T> {
  const res = await request<ApiResponse<T>>(path, {
    method: 'POST',
    data,
  });
  return unwrap(res);
}

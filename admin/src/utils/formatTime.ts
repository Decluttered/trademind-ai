import dayjs from 'dayjs';

/** Unified admin-panel date/time display format */
export const DATETIME_FORMAT = 'YYYY-MM-DD HH:mm:ss';

export type DateTimeInput = string | number | Date | null | undefined;

/** Formats an ISO string / timestamp into a locally readable time; returns fallback for invalid or empty values (default —) */
export function formatDateTime(value: DateTimeInput, fallback = '—'): string {
  if (value === null || value === undefined || value === '') return fallback;
  const d = dayjs(value);
  return d.isValid() ? d.format(DATETIME_FORMAT) : String(value);
}

/** @deprecated Use formatDateTime */
export const formatTs = formatDateTime;

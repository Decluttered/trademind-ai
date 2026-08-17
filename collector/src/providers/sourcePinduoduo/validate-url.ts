import { classifyPinduoduoUrl, isPifaWholesaleHost } from './url-type.js';

const PDD_HOSTS = new Set([
  'yangkeduo.com',
  'mobile.yangkeduo.com',
  'pinduoduo.com',
  'mobile.pinduoduo.com',
]);

export function isPinduoduoHost(hostname: string): boolean {
  const h = hostname.trim().toLowerCase();
  if (PDD_HOSTS.has(h)) return true;
  if (isPifaWholesaleHost(h)) return true;
  return h.endsWith('.yangkeduo.com') || h.endsWith('.pinduoduo.com');
}

/** Semantic validation: http(s) + Pinduoduo domain + recognizable wholesale detail path (mobile detail links are accepted but explicitly return unsupported). */
export function validatePinduoduoUrl(urlStr: string): boolean {
  const urlType = classifyPinduoduoUrl(urlStr);
  return urlType === 'wholesale_detail';
}

export function normalizePinduoduoNavUrl(raw: string): string {
  const trimmed = raw.trim();
  const urlType = classifyPinduoduoUrl(trimmed);
  if (urlType === 'wholesale_detail') {
    return trimmed;
  }
  try {
    const u = new URL(trimmed);
    const goodsId = u.searchParams.get('goods_id') ?? u.searchParams.get('goodsId');
    if (goodsId && /^\d+$/.test(goodsId)) {
      return `https://mobile.yangkeduo.com/goods.html?goods_id=${goodsId}`;
    }
  } catch {
    /* keep raw */
  }
  return trimmed;
}

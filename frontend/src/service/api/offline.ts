/**
 * 离线兜底：包装一个返回 `{ data, error }` 的请求函数
 *
 * - 成功：把结果写进 IndexedDB
 * - 失败且离线：返回上次缓存，并标记 `cached: true`
 */
import { loadCache, saveCache } from '@/utils/offline-cache';

interface FlatResult<T> {
  data: T | null;
  error: unknown;
}

export interface CachedResult<T> extends FlatResult<T> {
  /** 数据来自本地缓存（说明当前没网） */
  cached?: boolean;
  /** 缓存写入时间 */
  cachedAt?: number;
}

export function withOfflineCache<T>(key: string, fetcher: () => Promise<FlatResult<T>>) {
  return async (): Promise<CachedResult<T>> => {
    const result = await fetcher();

    if (result.data) {
      void saveCache(key, result.data);

      return result;
    }

    const cached = await loadCache<T>(key);

    if (cached) {
      return { data: cached, error: null, cached: true };
    }

    return result;
  };
}

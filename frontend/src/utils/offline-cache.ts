/**
 * 离线数据缓存（IndexedDB）
 *
 * 只缓存 GET 的接口结果：网络失败时用上次成功的数据兜底，
 * 让手机断网时至少还能看到列表，而不是白屏。
 */
const DB_NAME = 'gz-offline';
const DB_VERSION = 1;
const STORE = 'api';

interface CachedEntry {
  key: string;
  data: unknown;
  savedAt: number;
}

let dbPromise: Promise<IDBDatabase | null> | null = null;

function openDb(): Promise<IDBDatabase | null> {
  if (dbPromise) return dbPromise;

  dbPromise = new Promise(resolve => {
    if (typeof indexedDB === 'undefined') {
      resolve(null);

      return;
    }

    const req = indexedDB.open(DB_NAME, DB_VERSION);

    req.onupgradeneeded = () => {
      const db = req.result;

      if (!db.objectStoreNames.contains(STORE)) db.createObjectStore(STORE, { keyPath: 'key' });
    };

    req.onsuccess = () => resolve(req.result);
    req.onerror = () => resolve(null);
  });

  return dbPromise;
}

/** 缓存时间，用于提示「数据来自几点」 */
export function readCacheTime(key: string): Promise<number> {
  return readCache(key).then(entry => entry?.savedAt ?? 0);
}

async function readCache(key: string): Promise<CachedEntry | null> {
  const db = await openDb();

  if (!db) return null;

  return new Promise(resolve => {
    const tx = db.transaction(STORE, 'readonly');
    const req = tx.objectStore(STORE).get(key);

    req.onsuccess = () => resolve((req.result as CachedEntry) ?? null);
    req.onerror = () => resolve(null);
  });
}

/** 写入一次成功的响应 */
export async function saveCache(key: string, data: unknown) {
  const db = await openDb();

  if (!db) return;

  const tx = db.transaction(STORE, 'readwrite');

  tx.objectStore(STORE).put({ key, data, savedAt: Date.now() } satisfies CachedEntry);
}

/** 读缓存（网络失败时兜底） */
export async function loadCache<T>(key: string): Promise<T | null> {
  const entry = await readCache(key);

  return (entry?.data as T) ?? null;
}

/** 清空缓存（退出登录时调用） */
export async function clearCache() {
  const db = await openDb();

  if (!db) return;

  db.transaction(STORE, 'readwrite').objectStore(STORE).clear();
}

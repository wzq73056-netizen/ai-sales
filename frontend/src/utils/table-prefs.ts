import { fetchTableViewList, saveTableView } from '@/service/api/lead';

/** 列偏好存在服务端时用的固定名称 */
const PREFS_NAME = '__columns';

/**
 * 读取列偏好（本地 localStorage 兜底，再用服务端覆盖）
 *
 * @param tableKey 表格标识，例如 customers / lead / inbox
 * @param localKey 本地存储键
 */
export async function loadColumnPrefs(tableKey: string, localKey: string): Promise<string[] | null> {
  let local: string[] | null = null;

  try {
    const raw = localStorage.getItem(localKey);

    if (raw) local = JSON.parse(raw);
  } catch {
    /* 忽略损坏的本地设置 */
  }

  try {
    const { data } = await fetchTableViewList(tableKey);
    const row = data?.find(item => item.Name === PREFS_NAME);

    if (row) return JSON.parse(row.Payload) as string[];
  } catch {
    /* 服务端不可用时用本地 */
  }

  return local;
}

/** 保存列偏好：本地立即生效，服务端异步同步 */
export function saveColumnPrefs(tableKey: string, localKey: string, columns: string[]) {
  try {
    localStorage.setItem(localKey, JSON.stringify(columns));
  } catch {
    /* 忽略写入失败 */
  }

  saveTableView({ tableKey, name: PREFS_NAME, payload: JSON.stringify(columns) }).catch(() => {});
}

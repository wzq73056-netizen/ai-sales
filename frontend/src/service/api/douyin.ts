import { request } from '../request';

export interface DouyinTask {
  CrawlType: number;
  Name: string;
  Id: number;
  DeviceId: number;
  Industry: string;
  Keywords: string;
  VideosPerKeyword: number;
  MaxDays: number;
  LimitN: number;
  City: string;
  Status: number;
  ScheduleType: string;
  Schedule: string;
  NextRunAt: string | null;
  StartedAt: string | null;
  FinishedAt: string | null;
  VideosCount: number;
  CommentsCount: number;
  LeadsCount: number;
  Log: string;
  CreatedAt: string;
}

export interface TaskForm {
  CrawlType: number;
  DeviceId: number;
  Name: string;
  Industry: string;
  Keywords: string;
  VideosPerKeyword: number;
  MaxDays: number;
  LimitN: number;
  City: string;
  ExcludeWords: string;
  ScheduleType: 'manual' | 'daily' | 'interval';
  Schedule: string;
  AutoStart: boolean;
}

/** AI 依据行业生成关键词 */
export async function fetchGenKeywords(industry: string) {
  const { data, error } = await request<string[]>({
    url: '/lead/genKeywords',
    method: 'post',
    data: { Industry: industry }
  });

  if (error || !data) return { data: [] as string[], error };
  return { data, error: null };
}

/** 新建任务 */
export function fetchCreateDouyinTask(data: TaskForm) {
  return request<DouyinTask>({ url: '/lead/createDouyinTask', method: 'post', data });
}

/** 任务列表 */
export async function fetchDouyinTaskList(params: { page: number; pageSize: number }) {
  const { data, error } = await request<{ list: DouyinTask[]; total: number; page: number; pageSize: number }>({
    url: '/lead/getDouyinTaskList',
    method: 'get',
    params
  });

  if (error || !data) return { data: null, error };

  return {
    data: { records: data.list, current: data.page, size: data.pageSize, total: data.total },
    error: null
  };
}

/** 启动任务 */
export function fetchStartDouyinTask(id: number) {
  return request<null>({ url: '/lead/startDouyinTask', method: 'post', data: { Id: id } });
}

/** 停止当前任务 */
export function fetchStopDouyinTask() {
  return request<null>({ url: '/lead/stopDouyinTask', method: 'post' });
}

/** 删除任务 */
export function fetchDeleteDouyinTask(id: number) {
  return request<null>({ url: '/lead/deleteDouyinTask', method: 'delete', params: { Id: id } });
}

/** 当前运行状态 + 日志 */
export async function fetchRunningLog() {
  const { data, error } = await request<{ RunningTaskId: number; Log: string[] }>({
    url: '/lead/runningLog',
    method: 'get'
  });

  if (error || !data) return { data: null, error };
  return { data, error: null };
}

/** 抖音线索总数 */
export async function fetchDouyinLeadTotal() {
  const { data, error } = await request<{ list: unknown[]; total: number }>({
    url: '/lead/getLeadList',
    method: 'get',
    params: { page: 1, pageSize: 1, source: 'douyin' }
  });

  if (error || !data) return { data: 0, error };
  return { data: data.total ?? 0, error: null };
}

export interface DouyinAccount {
  Id: number;
  Keyword: string;
  Nickname: string;
  DouyinId: string;
  Fans: number;
  Bio: string;
  ProfileUrl: string;
  IsCompetitor: number;
  Remark: string;
  FollowAt: string | null;
  Score: number;
  Status: number;
  CreatedAt: string;
}

/** 账号线索列表 */
export async function fetchDouyinAccountList(params: {
  page: number;
  pageSize: number;
  keyword?: string;
  competitor?: number;
  sortBy?: string;
}) {
  const { data, error } = await request<{ list: DouyinAccount[]; total: number; page: number; pageSize: number }>({
    url: '/lead/getDouyinAccountList',
    method: 'get',
    params
  });

  if (error || !data) return { data: null, error };

  return {
    data: { records: data.list, current: data.page, size: data.pageSize, total: data.total },
    error: null
  };
}

/** 删除账号线索 */
export function fetchDeleteDouyinAccount(id: number) {
  return request<null>({ url: '/lead/deleteDouyinAccount', method: 'delete', params: { Id: id } });
}

/** 更新账号线索状态（可带跟进备注） */
export function fetchUpdateAccountStatus(id: number, status: number, remark?: string) {
  return request<null>({
    url: '/lead/updateDouyinAccountStatus',
    method: 'put',
    data: { Id: id, Status: status, Remark: remark ?? '' }
  });
}

/** 拉全量账号用于导出（最多 1000 条） */
export async function fetchAllDouyinAccounts(params: { keyword?: string; competitor?: number; sortBy?: string }) {
  const { data, error } = await request<{ list: DouyinAccount[]; total: number }>({
    url: '/lead/getDouyinAccountList',
    method: 'get',
    params: { ...params, page: 1, pageSize: 1000 }
  });

  if (error || !data) return { data: [] as DouyinAccount[], error };
  return { data: data.list ?? [], error: null };
}
export interface DouyinDm {
  Id: number;
  ProfileUrl: string;
  Nickname: string;
  DouyinId: string;
  Message: string;
  Status: number;
  ReplyStatus: number;
  ReplyText: string;
  ReplyAt: string | null;
  SentAt: string | null;
  CreatedAt: string | null;
}

/** 私信记录列表 */
export async function fetchDouyinDmList(params: { page: number; pageSize: number }) {
  const { data, error } = await request<{ list: DouyinDm[]; total: number; page: number; pageSize: number }>({
    url: '/lead/getDouyinDmList',
    method: 'get',
    params
  });

  if (error || !data) return { data: null, error };

  return {
    data: { records: data.list, current: data.page, size: data.pageSize, total: data.total },
    error: null
  };
}
export interface DmControl {
  Running: boolean;
  Listening: boolean;
  AutoReply: boolean;
  AutoSend: boolean;
  Rounds: number;
  Conversations: number;
  Changes: number;
  LastCheckAt: string;
  LastError: string;
  LastAutoReply: string;
}

/** 私信自动化状态 */
export async function fetchDmControl() {
  const { data, error } = await request<DmControl>({ url: '/lead/getDmControl', method: 'get' });

  if (error || !data) return { data: null, error };
  return { data, error: null };
}

/** 保存私信自动化开关 */
export async function saveDmControl(payload: {
  listen: boolean;
  autoReply: boolean;
  autoSend: boolean;
  interval: number;
}) {
  const { data, error } = await request<DmControl>({ url: '/lead/setDmControl', method: 'post', data: payload });

  if (error || !data) return { data: null, error };
  return { data, error: null };
}
/** 抖音采集账号（一个账号 = 一个浏览器） */
export interface DouyinDevice {
  Id: number;
  Name: string;
  DouyinId: string;
  ProfileDir: string;
  BrowserPath: string;
  BrowserName: string;
  CdpPort: number;
  Status: number;
  LastSeen: string | null;
}

/** 私信白名单 */
export interface DouyinWhitelist {
  Id: number;
  DeviceId: number;
  Nickname: string;
  DouyinId: string;
  ProfileUrl: string;
  Enabled: number;
}

export function fetchDouyinDevices() {
  return request<{ list: DouyinDevice[] }>({ url: '/lead/getDouyinDevices', method: 'get' });
}

export function saveDouyinDevice(data: Partial<DouyinDevice>) {
  return request<DouyinDevice>({ url: '/lead/saveDouyinDevice', method: 'post', data });
}

export function deleteDouyinDevice(id: number) {
  return request<null>({ url: '/lead/deleteDouyinDevice', method: 'post', data: { Id: id } });
}

export function fetchDouyinWhitelist(deviceId?: number) {
  return request<{ list: DouyinWhitelist[] }>({
    url: '/lead/getDouyinWhitelist',
    method: 'get',
    params: deviceId ? { deviceId } : {}
  });
}

export function saveDouyinWhitelist(data: Partial<DouyinWhitelist>) {
  return request<DouyinWhitelist>({ url: '/lead/saveDouyinWhitelist', method: 'post', data });
}

export function deleteDouyinWhitelist(id: number) {
  return request<null>({ url: '/lead/deleteDouyinWhitelist', method: 'post', data: { Id: id } });
}
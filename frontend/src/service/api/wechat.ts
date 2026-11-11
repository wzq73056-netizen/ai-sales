import { withOfflineCache } from './offline';
import { request } from '../request';

/** 微信转人工工单 */
export interface WxHandoff {
  Id: number;
  ChatId: string;
  Channel: string;
  NickName: string;
  Category: string;
  Reason: string;
  LastMsg: string;
  Contact: string;
  WechatAdded: number;
  Result: string;
  DraftReply: string;
  Urgency: number;
  Status: number;
  AiPaused: number;
  ClaimUser: string;
  ClaimAt: string | null;
  CreatedAt: string | null;
}

/** 待接管列表 */
export async function fetchWxHandoffList(params: { page: number; pageSize: number; status: number }) {
  const { data, error } = await request<{ list: WxHandoff[]; total: number }>({
    url: '/lead/getWxHandoffList',
    method: 'get',
    params
  });

  if (error || !data) return { data: null, error };

  return { data: { records: data.list, total: data.total }, error: null };
}

/** 认领工单 */
export function claimWxHandoff(id: number, user: string) {
  return request<null>({ url: '/lead/claimWxHandoff', method: 'post', data: { Id: id, User: user } });
}

/** 处理 / 忽略 */
export function resolveWxHandoff(id: number, status: number) {
  return request<null>({ url: '/lead/resolveWxHandoff', method: 'post', data: { Id: id, Status: status } });
}

/** 跟进记录 */
export interface WxFollowup {
  Id: number;
  NickName: string;
  ChatId: string;
  Kind: string;
  Content: string;
  MediaUrl?: string;
  Operator: string;
  CreatedAt: string | null;
}

/** 客户详情 */
export interface WxCustomerDetail {
  Customer: WxCustomer;
  Followups: WxFollowup[];
}

/** 客户列表（按昵称合并） */
export function fetchWxCustomerGrouped(params: { minScore: number }) {
  return request<{ list: WxCustomer[]; total: number }>({
    url: '/lead/getWxCustomerGrouped',
    method: 'get',
    params
  });
}

/** 客户详情 */
export function fetchWxCustomerDetail(nick: string) {
  return request<WxCustomerDetail>({ url: '/lead/getWxCustomerDetail', method: 'get', params: { nick } });
}

/** 新增跟进记录 */
export function addWxFollowup(data: { NickName: string; ChatId: string; Kind: string; Content: string; Operator: string }) {
  return request<null>({ url: '/lead/addWxFollowup', method: 'post', data });
}

/** 微信客户跟进价值 */
export interface WxCustomer {
  Id: number;
  Side?: string;
  ChatId: string;
  Channel: string;
  NickName: string;
  Score: number;
  Tier: string;
  Stage: string;
  Tags: string;
  MyCount: number;
  TheirCount: number;
  LastMsgAt: string | null;
  SilentDays: number;
  FollowStatus: number;
  NextFollowAt: string | null;
  Reason: string;
  LlmScore: number;
  LlmReason: string;
  NextAction: string;
}

export function fetchWxCustomerList(params: { page: number; pageSize: number; minScore?: number; tier?: string; stage?: string; keyword?: string; side?: string; sort?: string; order?: string }) {
  const key = `customers:${params.page}:${params.pageSize}:${params.tier || ''}:${params.stage || ''}:${params.keyword || ''}`;

  return withOfflineCache(key, async () => {
    const { data, error } = await request<{ list: WxCustomer[]; total: number }>({
      url: '/lead/getWxCustomerList',
      method: 'get',
      params
    });

    if (error || !data) return { data: null, error };

    return { data: { records: data.list, total: data.total }, error: null };
  })();
}

export function markWxCustomerFollow(id: number, status: number) {
  return request<null>({ url: '/lead/markWxCustomerFollow', method: 'post', data: { Id: id, Status: status } });
}
/** 更新转微信跟进状态 */
export function updateWxHandoffResult(id: number, wechatAdded: number, result: string) {
  return request<null>({
    url: '/lead/updateWxHandoffResult',
    method: 'post',
    data: { Id: id, WechatAdded: wechatAdded, Result: result }
  });
}
/** 手动恢复 AI 自动回复 */
export function resumeAi(id: number) {
  return request<null>({ url: '/lead/resumeAi', method: 'post', data: { Id: id } });
}
/** 转化漏斗 */
export interface FunnelStage {
  name: string;
  count: number;
  rate: number;
}

export interface FunnelCategory {
  name: string;
  count: number;
}

export interface FunnelDay {
  date: string;
  customers: number;
  handoffs: number;
}

export interface FunnelStats {
  total: number;
  stages: FunnelStage[];
  categories: FunnelCategory[];
  trend: FunnelDay[];
  dealCount: number;
  dealRate: number;
  avgScore: number;
  handoffTotal: number;
}

export function fetchFunnelStats() {
  return request<FunnelStats>({ url: '/lead/getFunnelStats', method: 'get' });
}
export interface ScriptStat {
  Script: string;
  Platform: string;
  Sent: number;
  Replied: number;
  Rate: number;
}

export interface ScriptStats {
  Total: number;
  Replied: number;
  Rate: number;
  Items: ScriptStat[];
}

export function fetchScriptStats() {
  return request<ScriptStats>({ url: '/lead/getScriptStats', method: 'get' });
}
export interface WxAddConfig {
  Id: number;
  Enabled: number;
  DailyLimit: number;
  IntervalMin: number;
  IntervalMax: number;
  PauseMinutes: number;
  WindowStart: string;
  WindowEnd: string;
  Remark: string;
  PausedUntil: string;
  LastRunAt: string;
}

export interface WxAddQueueRow {
  Id: number;
  LeadId: number;
  Contact: string;
  ContactName: string;
  Remark: string;
  Status: number;
  Reason: string;
  ExecutedAt: string;
}

export interface WxAddStats {
  todayOk: number;
  todayFail: number;
  todayTotal: number;
  pending: number;
  dailyLimit: number;
}

export function fetchWxAddConfig() {
  return request<{ config: WxAddConfig; stats: WxAddStats }>({ url: '/lead/getWxAddConfig', method: 'get' });
}

export function saveWxAddConfig(data: WxAddConfig) {
  return request<null>({ url: '/lead/saveWxAddConfig', method: 'post', data });
}

export function fetchWxAddQueue() {
  return request<{ list: WxAddQueueRow[]; stats: WxAddStats }>({ url: '/lead/getWxAddQueue', method: 'get' });
}

export function runWxAddOnce() {
  return request<{ pending: number; note: string }>({ url: '/lead/runWxAddOnce', method: 'post', data: {} });
}
export interface ReminderRow {
  Id: number;
  NickName: string;
  Tier: string;
  Stage: string;
  Score: number;
  NextAction: string;
  NextFollow: string;
  SilentDays: number;
  LastMsgAt: string;
  Bucket: string;
  Reason: string;
}

export interface ReminderBoard {
  Overdue: ReminderRow[];
  Today: ReminderRow[];
  Week: ReminderRow[];
  Silent: ReminderRow[];
  Total: number;
}

export function fetchFollowupReminders() {
  return withOfflineCache('reminders', () => request<ReminderBoard>({ url: '/lead/getFollowupReminders', method: 'get' }))();
}
export function updateWxCustomer(data: { Id: number; Stage?: string; NextFollowAt?: string; NextAction?: string; Tags?: string }) {
  return request<null>({ url: '/lead/updateWxCustomer', method: 'post', data });
}
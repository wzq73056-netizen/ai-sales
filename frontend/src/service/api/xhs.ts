import { request } from '../request';

export interface XhsDevice {
  Id: number;
  Platform: string;
  Name: string;
  DouyinId: string;
  ProfileDir: string;
  BrowserPath: string;
  BrowserName: string;
  CdpPort: number;
  Status: number;
}

export interface XhsWhitelist {
  Id: number;
  Platform: string;
  DeviceId: number;
  Nickname: string;
  DouyinId: string;
  ProfileUrl: string;
  Enabled: number;
}

export interface XhsCollectStatus {
  running: boolean;
  started: string;
  finished: string;
  log: string;
}

export interface XhsLeadStats {
  total: number;
  today: number;
  grades: { lv: string; count: number }[];
}

export function fetchXhsDevices() {
  return request<{ list: XhsDevice[] }>({ url: '/lead/getDouyinDevices', method: 'get', params: { platform: 'xhs' } });
}

export function saveXhsDevice(data: Partial<XhsDevice>) {
  return request<XhsDevice>({ url: '/lead/saveDouyinDevice', method: 'post', data: { ...data, Platform: 'xhs' } });
}

export function fetchXhsWhitelist() {
  return request<{ list: XhsWhitelist[] }>({
    url: '/lead/getDouyinWhitelist',
    method: 'get',
    params: { platform: 'xhs' }
  });
}

export function saveXhsWhitelist(data: Partial<XhsWhitelist>) {
  return request<XhsWhitelist>({ url: '/lead/saveDouyinWhitelist', method: 'post', data: { ...data, Platform: 'xhs' } });
}

export function deleteXhsWhitelist(id: number) {
  return request<null>({ url: '/lead/deleteDouyinWhitelist', method: 'post', data: { Id: id } });
}

export function startXhsCollect(data: { Keywords: string; Notes: number; Limit: number }) {
  return request<XhsCollectStatus>({ url: '/lead/startXhsCollect', method: 'post', data });
}

export function fetchXhsCollectStatus() {
  return request<XhsCollectStatus>({ url: '/lead/getXhsCollectStatus', method: 'get' });
}

export function fetchXhsLeadStats() {
  return request<XhsLeadStats>({ url: '/lead/getXhsLeadStats', method: 'get' });
}

export interface XhsRun {
  Id: number;
  Keywords: string;
  Notes: number;
  NotesDone: number;
  CommentsKept: number;
  Status: number;
  Message: string;
  LogTail: string;
  StartedAt: string;
  FinishedAt: string;
}

export function fetchXhsRuns() {
  return request<{ list: XhsRun[] }>({ url: '/lead/getXhsRuns', method: 'get' });
}

export function sendXhsDm(data: { NickName: string; Message?: string; AiHint?: string; Send: boolean }) {
  return request<{ ok: boolean; output?: string; error?: string }>({
    url: '/lead/sendXhsDm',
    method: 'post',
    data
  });
}
export function genXhsScripts(limit = 8) {
  return request<{ generated: number }>({
    url: '/lead/genLeadDmScripts',
    method: 'post',
    data: { Grade: 'B', Limit: limit, Source: 'xhs' }
  });
}
export interface XhsSchedule {
  Id: number;
  Keywords: string;
  Notes: number;
  CommentLimit: number;
  RunAt: string;
  Enabled: number;
  LastRunDate: string;
}

export function fetchXhsSchedule() {
  return request<XhsSchedule>({ url: '/lead/getXhsSchedule', method: 'get' });
}

export function saveXhsSchedule(data: XhsSchedule) {
  return request<null>({ url: '/lead/saveXhsSchedule', method: 'post', data });
}
export function fetchXhsLeads() {
  return request<{ list: { Id: number; ContactName: string; Remark: string }[] }>({
    url: '/lead/getLeadList',
    method: 'get',
    params: { page: 1, pageSize: 100, source: 'xhs' }
  });
}

export function previewXhsLeadDm(data: { LeadId: number; Message?: string; AiHint?: string }) {
  return request<{ ok: boolean; output?: string; error?: string }>({
    url: '/lead/sendXhsProactiveDm',
    method: 'post',
    data: { ...data, Send: false }
  });
}
export function sendXhsLeadDm(data: { LeadId: number; Message?: string; AiHint?: string }) {
  return request<{ ok: boolean; output?: string; error?: string }>({
    url: '/lead/sendXhsProactiveDm',
    method: 'post',
    data: { ...data, Send: true }
  });
}
export function genLeadDmScript(id: number) {
  return request<{ script: string }>({ url: '/lead/genLeadDmScript', method: 'post', data: { Id: id } });
}
export interface ScrapeKeyword {
  Id: number;
  Platform: string;
  Keyword: string;
  Enabled: number;
}

export interface ScrapeBlacklist {
  Id: number;
  Kind: string;
  Value: string;
  Reason: string;
}

export function fetchKeywords(platform: string) {
  return request<{ list: ScrapeKeyword[] }>({ url: '/lead/getScrapeKeywords', method: 'get', params: { platform } });
}

export function saveKeyword(data: { Platform: string; Keyword: string; Enabled: number }) {
  return request<null>({ url: '/lead/saveScrapeKeyword', method: 'post', data });
}

export function deleteKeyword(id: number) {
  return request<null>({ url: '/lead/deleteScrapeKeyword', method: 'post', data: { Id: id } });
}

export function fetchBlacklist() {
  return request<{ list: ScrapeBlacklist[] }>({ url: '/lead/getScrapeBlacklist', method: 'get' });
}

export function saveBlacklist(data: { Kind: string; Value: string; Reason: string }) {
  return request<null>({ url: '/lead/saveScrapeBlacklist', method: 'post', data });
}

export function deleteBlacklist(id: number) {
  return request<null>({ url: '/lead/deleteScrapeBlacklist', method: 'post', data: { Id: id } });
}
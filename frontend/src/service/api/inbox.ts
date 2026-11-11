import { request } from '../request';

export interface InboxItem {
  kind: 'handoff' | 'message';
  Channel: string;
  Id: number;
  Name: string;
  Text: string;
  Time: string;
  Urgency: number;
  Status: number;
  Tag: string;
}

export interface InboxStats {
  handoffCount: number;
  messageCount: number;
  wechatCount: number;
  douyinCount: number;
  xhsCount: number;
}

export interface Inbox {
  stats: InboxStats;
  items: InboxItem[];
}

export function fetchInbox(channel?: string) {
  return request<Inbox>({ url: '/lead/getInbox', method: 'get', params: channel ? { channel } : {} });
}

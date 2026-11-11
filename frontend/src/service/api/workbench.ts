import { request } from '../request';

/** 工作台待办 */
export interface WorkbenchItem {
  kind: string;
  title: string;
  subtitle: string;
  count: number;
  action: string;
}

export interface WorkbenchLead {
  Id: number;
  Company: string;
  Keyword: string;
  City: string;
  Score: number;
  Grade: string;
  Ready: number;
  Source: string;
  CreatedAt: string;
}

export interface WorkbenchCustomer {
  Id: number;
  NickName: string;
  Tier: string;
  Stage: string;
  Score: number;
  SilentDays: number;
  NextAction: string;
}

export interface WorkbenchHandoff {
  Id: number;
  Channel: string;
  NickName: string;
  Category: string;
  LastMsg: string;
  Urgency: number;
}

export interface Workbench {
  today: string;
  handoffCount: number;
  hotCount: number;
  silentCount: number;
  leadCount: number;
  newLeadToday: number;
  replyToday: number;
  items: WorkbenchItem[];
  handoffs: WorkbenchHandoff[];
  hotCustomers: WorkbenchCustomer[];
  silentCustomers: WorkbenchCustomer[];
  newLeads: WorkbenchLead[];
  judgement?: { headline: string; basis: string[]; suggestion: string; impact: string };
}

export function fetchWorkbench() {
  return request<Workbench>({ url: '/lead/getWorkbench', method: 'get' });
}

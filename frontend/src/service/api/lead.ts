import { request } from '../request';

interface GvaList<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface LeadRow {
  Id: number;
  Company: string;
  ContactName: string | null;
  Phone: string | null;
  Wechat: string | null;
  Email: string | null;
  City: string | null;
  Industry: string | null;
  Scale: string | null;
  Source: string;
  SourceUrl: string | null;
  Keyword: string | null;
  Score: number;
  Grade: string;
  ContactReady: number;
  DmScript: string;
  Status: number;
  Owner: string | null;
  Remark: string | null;
  ExtractInfo: string | null;
  Avatar: string | null;
  VideoTitle: string | null;
  CreatedAt: string;
}

/** 线索列表 */
export async function fetchGetLeadList(params: { page: number; pageSize: number; status?: number; source?: string; side?: string; grade?: string; sort?: string; order?: string }) {
  const { data, error } = await request<GvaList<LeadRow>>({
    url: '/lead/getLeadList',
    method: 'get',
    params
  });

  if (error || !data) {
    return { data: null, error };
  }

  return {
    data: { records: data.list, current: data.page, size: data.pageSize, total: data.total },
    error: null
  };
}

/** 更新线索（主要用来改跟进状态） */
export function fetchUpdateLead(payload: Partial<LeadRow> & { Id: number }) {
  return request({ url: '/lead/updateLead', method: 'put', data: payload });
}

/** 删除线索 */
export function fetchDeleteLead(id: number) {
  return request({ url: '/lead/deleteLead', method: 'delete', params: { Id: id } });
}

/** 生成单条线索的私信话术 */
export function genLeadDmScript(id: number) {
  return request<{ script: string }>({ url: '/lead/genLeadDmScript', method: 'post', data: { Id: id } });
}

/** 批量生成话术 */
export function genLeadDmScripts(grade = 'A', limit = 10) {
  return request<{ generated: number }>({ url: '/lead/genLeadDmScripts', method: 'post', data: { Grade: grade, Limit: limit } });
}
export function convertLead(id: number) {
  return request<{ ok: boolean; duplicated: boolean; customerId: number }>({
    url: '/lead/convertLead',
    method: 'post',
    data: { Id: id }
  });
}

export async function exportLeads(params: { source?: string; grade?: string }) {
  const query = new URLSearchParams();
  if (params.source) query.set('source', params.source);
  if (params.grade) query.set('grade', params.grade);

  const res = await fetch(`/proxy-default/lead/exportLeads?${query.toString()}`, {
    headers: { 'x-token': localStorage.getItem('token') || '' }
  });
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `leads-${new Date().toISOString().slice(0, 10)}.xlsx`;
  a.click();
  URL.revokeObjectURL(url);
}
export function addWxFriend(id: number, remark: string, send = true) {
  return request<{ ok: boolean; contact: string; output: string }>({
    url: '/lead/addWxFriend',
    method: 'post',
    data: { Id: id, Remark: remark, Send: send }
  });
}
export function convertLeads(ids: number[]) {
  return request<{ converted: number; duplicated: number; failed: number }>({
    url: '/lead/convertLeads',
    method: 'post',
    data: { Ids: ids }
  });
}

export function fetchLeadAnalytics() {
  return request<{
    sources: { Source: string; Total: number; GradeA: number; GradeB: number; GradeC: number; WithContact: number }[];
    health: {
      CustomerTotal: number;
      Overdue: number;
      OverdueRate: number;
      Silent7: number;
      Silent30: number;
      AvgSilentDays: number;
      DealCount: number;
      ValidCount: number;
    };
  }>({ url: '/lead/getLeadAnalytics', method: 'get' });
}
/* ---------- 表格视图（服务端存储） ---------- */
export interface TableViewRow {
  Id: number;
  TableKey: string;
  Name: string;
  Payload: string;
  CreatedAt: number;
  UpdatedAt: number;
}

export function fetchTableViewList(tableKey: string) {
  return request<TableViewRow[]>({
    url: '/lead/tableViewList',
    method: 'get',
    params: { tableKey }
  });
}

export function saveTableView(data: { tableKey: string; name: string; payload: string }) {
  return request({
    url: '/lead/tableViewSave',
    method: 'post',
    data
  });
}

export function deleteTableView(tableKey: string, name: string) {
  return request({
    url: '/lead/tableViewDelete',
    method: 'delete',
    params: { tableKey, name }
  });
}
import { request } from '../request';

/** gin-vue-admin 列表返回结构 */
interface GvaList<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}

/** ai_knowledge 行（生成器产出的字段名首字母大写） */
export interface KnowledgeRow {
  Id: number;
  Name: string;
  Description: string | null;
  Share: number;
  Status: number;
  RequestCount: number;
  CreatedAt: string;
  LegacyTable: string | null;
}

/** 新增/编辑提交体（与 Go struct 的 json tag 对齐，首字母大写） */
export interface KnowledgePayload {
  Id?: number;
  UserId?: number;
  Name: string;
  Description: string;
  Share: number;
  Status: number;
  RequestCount?: number;
  CreatedAt?: string;
  UpdatedAt?: string;
}

/**
 * 知识库列表
 *
 * 把 gin-vue-admin 的 { list, total, page, pageSize } 转成 soybean 习惯的
 * { records, current, size, total }，这样表格 hook 可以直接用。
 */
export async function fetchGetKnowledgeList(params: { page: number; pageSize: number }) {
  const { data, error } = await request<GvaList<KnowledgeRow>>({
    url: '/kb/getKnowledgeList',
    method: 'get',
    params
  });

  if (error || !data) {
    return { data: null, error };
  }

  return {
    data: {
      records: data.list,
      current: data.page,
      size: data.pageSize,
      total: data.total
    },
    error: null
  };
}

/** 新增知识库 */
export function fetchCreateKnowledge(payload: KnowledgePayload) {
  return request({ url: '/kb/createKnowledge', method: 'post', data: payload });
}

/** 编辑知识库 */
export function fetchUpdateKnowledge(payload: KnowledgePayload & { Id: number }) {
  return request({ url: '/kb/updateKnowledge', method: 'put', data: payload });
}

/** 删除知识库（生成器的 handler 读的是 query 参数 `Id`，不是 `ID`） */
export function fetchDeleteKnowledge(id: number) {
  return request({ url: '/kb/deleteKnowledge', method: 'delete', params: { Id: id } });
}

/** 每个知识库的分片数量（卡片上显示） */
export async function fetchGetKnowledgeStats() {
  const { data, error } = await request<{ KnowledgeId: number; FragmentCount: number }[]>({
    url: '/kb/getKnowledgeStats',
    method: 'get'
  });

  if (error || !data) {
    return { data: {} as Record<number, number>, error };
  }

  const map: Record<number, number> = {};
  data.forEach(item => {
    map[item.KnowledgeId] = item.FragmentCount;
  });

  return { data: map, error: null };
}

/** 批量删除知识库（handler 读的是 `Ids[]`，axios 默认数组序列化正好是这个格式） */
export function fetchDeleteKnowledgeByIds(ids: number[]) {
  return request({ url: '/kb/deleteKnowledgeByIds', method: 'delete', params: { Ids: ids } });
}

/** ai_knowledge_fragment 行 */
export interface FragmentRow {
  Id: number;
  KnowledgeId: number;
  Idx: number;
  DocId: string | null;
  Content: string;
  CreatedAt: string;
}

/** 分片列表（按所属知识库过滤，父子表关联） */
export async function fetchGetFragmentList(params: { knowledgeId: number; page: number; pageSize: number }) {
  const { data, error } = await request<GvaList<FragmentRow>>({
    url: '/kbfrag/getKnowledgeFragmentList',
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

/** 新增分片 */
export function fetchCreateFragment(payload: {
  KnowledgeId: number;
  Idx: number;
  DocId: string;
  Content: string;
  CreatedAt: string;
  UpdatedAt: string;
}) {
  return request({ url: '/kbfrag/createKnowledgeFragment', method: 'post', data: payload });
}

/** 删除分片 */
export function fetchDeleteFragment(id: number) {
  return request({ url: '/kbfrag/deleteKnowledgeFragment', method: 'delete', params: { Id: id } });
}

/** 文档行 */
export interface KnowledgeDocument {
  Id: number;
  KnowledgeId: number;
  Name: string;
  Format: string;
  Size: number;
  ChunkCount: number;
  Status: number;
  CreatedAt: string;
}

/** 文档列表 */
export async function fetchGetDocumentList(params: { knowledgeId: number; page: number; pageSize: number }) {
  const { data, error } = await request<GvaList<KnowledgeDocument>>({
    url: '/kbfrag/getDocumentList',
    method: 'get',
    params
  });

  if (error || !data) {
    return { data: null, error };
  }

  return { data: { records: data.list, current: data.page, size: data.pageSize, total: data.total }, error: null };
}

/** 上传文档（后端自动切块入库） */
export function fetchUploadDocument(knowledgeId: number, file: File) {
  const form = new FormData();
  form.append('knowledgeId', String(knowledgeId));
  form.append('file', file);

  return request<KnowledgeDocument>({
    url: '/kbfrag/uploadDocument',
    method: 'post',
    data: form,
    headers: { 'Content-Type': 'multipart/form-data' }
  });
}

/** 删除文档（连带其片段） */
export function fetchDeleteDocument(id: number) {
  return request({ url: '/kbfrag/deleteDocument', method: 'delete', params: { Id: id } });
}

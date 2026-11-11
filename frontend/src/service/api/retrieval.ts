import { request } from '../request';

export interface SearchHit {
  Id: number;
  KnowledgeId: number;
  DocumentId: number;
  DocumentName: string;
  Idx: number;
  DocId: string;
  Content: string;
  Score: number;
  /** 0-100 的相似度（BM25 归一化或大模型打分） */
  Similarity: number;
}

/** 知识库检索（BM25 粗排 + 大模型精排） */
export function fetchSearchFragments(params: {
  knowledgeId: number;
  query: string;
  topK: number;
  minScore?: number;
}) {
  return request<SearchHit[]>({
    url: '/kbfrag/searchFragments',
    method: 'post',
    data: params
  });
}

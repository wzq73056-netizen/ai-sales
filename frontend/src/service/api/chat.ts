import { request } from '../request';
import type { SearchHit } from './retrieval';

export interface ChatResult {
  /** retrieval-only：只检索没生成；llm：大模型生成 */
  Mode: 'retrieval-only' | 'llm';
  Answer: string;
  Prompt: string;
  Sources: SearchHit[];
}

/** 知识库问答（RAG） */
export function fetchChat(params: { knowledgeId: number; question: string; topK: number }) {
  return request<ChatResult>({
    url: '/kbfrag/chat',
    method: 'post',
    data: params
  });
}

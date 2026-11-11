import { request } from '../request';

export interface LLMConfigForm {
  baseUrl: string;
  apiKey: string;
  model: string;
  kbIdBuyer: string;
  kbIdSeller: string;
  temperature: number;
}

export interface LLMConfigData extends LLMConfigForm {
  hasKey: boolean;
  filePath: string;
}

/** 读取大模型配置（key 已打码） */
export function fetchGetLLMConfig() {
  return request<LLMConfigData>({
    url: '/lead/getLLMConfig',
    method: 'get'
  });
}

/** 保存大模型配置 */
export function fetchSaveLLMConfig(data: LLMConfigForm) {
  return request({
    url: '/lead/saveLLMConfig',
    method: 'post',
    data
  });
}

/** 测试当前配置能否调通 */
export function fetchTestLLMConfig() {
  return request<{ reply: string; model: string }>({
    url: '/lead/testLLMConfig',
    method: 'post'
  });
}

import { request } from '../request';

export interface SystemAlert {
  Level: 'error' | 'warning' | 'info';
  Title: string;
  Detail: string;
  Action?: string;
  ActionName?: string;
}

export interface WxLoginStatus {
  Ok: boolean;
  FlagTime: string;
  LastSuccess: string;
  Message: string;
}

/** 系统告警：微信掉登录、监听心跳断开等 */
export function fetchSystemAlerts() {
  return request<{ list: SystemAlert[]; count: number; wx: WxLoginStatus }>({
    url: '/lead/getSystemAlerts',
    method: 'get'
  });
}

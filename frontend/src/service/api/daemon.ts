import { request } from '../request';

export interface DaemonItem {
  Key: string;
  Name: string;
  Desc: string;
  Alive: boolean;
  Pid: number;
  StartedAt: string;
  Heartbeat: string;
  LogFile: string;
  LastLog: string;
}

/** 三个监听进程的当前状态 */
export function fetchDaemons() {
  return request<{ list: DaemonItem[] }>({
    url: '/lead/getDaemons',
    method: 'get'
  });
}

/** 启动某个监听进程 */
export function startDaemon(key: string) {
  return request<{ list: DaemonItem[]; msg: string }>({
    url: '/lead/startDaemon',
    method: 'post',
    data: { key }
  });
}

/** 停止某个监听进程 */
export function stopDaemon(key: string) {
  return request<{ list: DaemonItem[]; msg: string }>({
    url: '/lead/stopDaemon',
    method: 'post',
    data: { key }
  });
}

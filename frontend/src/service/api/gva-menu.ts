import { request } from '../request';

interface GvaMenuNode {
  name: string;
  path?: string;
  title?: string;
  children?: GvaMenuNode[];
}

/** 递归收集菜单 name 和 path（path 用来兜底匹配改过名的子路由） */
function collectNames(nodes: GvaMenuNode[], out: string[]) {
  nodes.forEach(node => {
    if (node.name) out.push(node.name);
    if (node.path) out.push(node.path);
    if (node.children?.length) collectNames(node.children, out);
  });
}

/**
 * 取当前登录用户可见的菜单（gin-vue-admin 的 /menu/getMenu 是 POST）
 *
 * 返回菜单 name 列表，前端用它对路由做白名单过滤。
 */
export async function fetchAllowedMenuNames() {
  const { data, error } = await request<{ menus: GvaMenuNode[] }>({
    url: '/menu/getMenu',
    method: 'post',
    data: {}
  });

  if (error || !data) {
    return { data: null, error };
  }

  const names: string[] = [];
  collectNames(data.menus ?? [], names);

  return { data: names, error: null };
}

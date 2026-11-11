import type { CustomRoute, ElegantConstRoute, LastLevelRouteKey, RouteKey } from '@elegant-router/types';
import { request } from '../request';

interface GvaMenu {
  name: string;
  path: string;
  component?: string;
  title?: string;
  icon?: string;
  hidden?: number;
  sort?: number;
  /** gin 的标题/图标都在 meta 里，顶层没有 */
  meta?: {
    title?: string;
    icon?: string;
    keepAlive?: boolean;
    activeName?: string;
    closeTab?: boolean;
  };
  children?: GvaMenu[];
}

/** `view/knowledge/index.vue` → `view.knowledge`，与 elegant-router 的组件表对齐 */
function toComponentKey(component?: string) {
  if (!component) return '';

  return component
    .replace(/^view\//, '')
    .replace(/\/index\.vue$/, '')
    .replace(/\.vue$/, '')
    .replace(/\//g, '_');
}

function toRoute(menu: GvaMenu): ElegantConstRoute {
  const meta: Record<string, any> = {
    title: menu.title || menu.name,
    i18nKey: `route.${menu.name}`,
    order: menu.sort ?? 1,
    hideInMenu: menu.hidden === 1
  };

  if (menu.icon) meta.icon = menu.icon;

  const key = toComponentKey(menu.component);

  const route = {
    name: menu.name,
    path: menu.path,
    // soybean 用的是「布局$视图」的扁平写法，不是嵌套 children
    component: `layout.base$view.${key}`,
    meta
  } as unknown as ElegantConstRoute;

  return route;
}

/** 常量路由（login / 403 / 404 / 500）：走静态，失败时 route store 会自动回退 */
export function fetchGetConstantRoutes() {
  return request<CustomRoute[]>({ url: '/route/getConstantRoutes' });
}

/**
 * 用户路由：由 gin-vue-admin 的菜单表驱动
 *
 * 返回结构必须符合 soybean 的 `{ routes, home }`；顶层包一层 layout.base，
 * 否则侧边栏渲染不出来。
 */
export async function fetchGetUserRoutes() {
  const { data, error } = await request<{ menus: GvaMenu[] }>({
    url: '/menu/getMenu',
    method: 'post',
    data: {}
  });

  if (error || !data) {
    return { data: null, error: error ?? new Error('get menu failed') };
  }

  const menus = data.menus ?? [];

  if (!menus.length) {
    return { data: { routes: [] as ElegantConstRoute[], home: 'home' as LastLevelRouteKey }, error: null };
  }

  /**
   * 完全按后端菜单树生成路由：
   * - 有 children 的节点 = 目录，包 `layout.base`
   * - 叶子节点 = 页面。顶层叶子用 `layout.base$view.x` 的扁平写法；
   *   子节点不能用布局前缀，且**路由名必须带 `_`**（soybean 用下划线判断亲子关系）
   */
  function metaOf(menu: GvaMenu, extra: Record<string, any> = {}) {
    const title = menu.meta?.title || menu.title || menu.name;
    const icon = menu.meta?.icon || menu.icon;

    return {
      title,
      order: menu.sort ?? 1,
      hideInMenu: menu.hidden === 1 || (menu.hidden as unknown as boolean) === true,
      ...(icon ? { icon } : {}),
      ...extra
    };
  }

  function toLeaf(menu: GvaMenu, parentName?: string): ElegantConstRoute {
    const key = toComponentKey(menu.component);
    const name = parentName ? (menu.name.includes('_') ? menu.name : `${parentName}_${menu.name}`) : menu.name;

    return {
      name,
      path: menu.path,
      component: parentName ? `view.${key}` : `layout.base$view.${key}`,
      meta: metaOf(menu)
    } as unknown as ElegantConstRoute;
  }

  function toRoute(menu: GvaMenu, parentName?: string): ElegantConstRoute {
    const children = menu.children ?? [];

    if (!children.length) {
      return toLeaf(menu, parentName);
    }

    const parentKey = menu.name;
    const childRoutes = children.map(child => toRoute(child, parentKey));

    return {
      name: parentKey,
      path: menu.path || `/${parentKey}`,
      component: 'layout.base',
      redirect: childRoutes[0].path,
      meta: metaOf(menu),
      children: childRoutes
    } as unknown as ElegantConstRoute;
  }

  const routes = menus.map(menu => toRoute(menu));

  const home =
    menus.find(menu => !menu.children?.length)?.name ??
    menus.find(menu => menu.children?.length)?.['children']?.[0]?.name ??
    'home';

  return { data: { routes, home: home as LastLevelRouteKey }, error: null };
}

/** 取第一条可用的叶子路由名作为 home */
function children0Name(routes: ElegantConstRoute[]): string {
  for (const r of routes) {
    const children = (r as any).children;

    if (children?.length) return children[0].name;
  }

  return 'home';
}

/**
 * 路由是否存在
 *
 * gin 没有这个接口，直接按前端已有的组件表判断：能拿到 component 就说明页面存在。
 */
export async function fetchIsRouteExist(routeName: string) {
  const { data, error } = await request<{ menus: GvaMenu[] }>({
    url: '/menu/getMenu',
    method: 'post',
    data: {}
  });

  if (error || !data) {
    return { data: false, error };
  }

  const names: string[] = [];

  const walk = (list: GvaMenu[]) => {
    list.forEach(item => {
      names.push(item.name);
      if (item.children?.length) walk(item.children);
    });
  };

  walk(data.menus ?? []);

  return { data: names.includes(routeName), error: null };
}

import { request } from '../request';

/** 角色列表 —— 对接 gin-vue-admin POST /authority/getAuthorityList */
export async function fetchGetRoleList(params?: Api.SystemManage.RoleSearchParams) {
  const { data, error } = await request<any[]>({
    url: '/authority/getAuthorityList',
    method: 'post',
    data: { page: params?.current ?? 1, pageSize: params?.size ?? 10 }
  });

  if (error || !data) {
    return { data: null as unknown as Api.SystemManage.RoleList, error };
  }

  const records = data.map((a: any) => ({
    id: a.authorityId,
    roleName: a.authorityName,
    roleCode: String(a.authorityId),
    roleDesc: a.authorityName,
    menuIds: [],
    status: '1',
    createBy: '',
    createTime: a.CreatedAt ?? '',
    updateBy: '',
    updateTime: a.UpdatedAt ?? ''
  }));

  return {
    data: {
      records,
      current: params?.current ?? 1,
      size: params?.size ?? 10,
      total: records.length
    } as unknown as Api.SystemManage.RoleList,
    error: null
  };
}

/**
 * get all roles —— 角色下拉用同一个接口
 */
export async function fetchGetAllRoles() {
  const { data, error } = await request<any[]>({
    url: '/authority/getAuthorityList',
    method: 'post',
    data: { page: 1, pageSize: 100 }
  });

  if (error || !data) {
    return { data: [] as Api.SystemManage.AllRole[], error };
  }

  return {
    data: data.map((a: any) => ({ roleName: a.authorityName, roleCode: String(a.authorityId) })) as unknown as Api.SystemManage.AllRole[],
    error: null
  };
}

/** get user list —— 对接 gin-vue-admin POST /user/getUserList */
export async function fetchGetUserList(params?: Api.SystemManage.UserSearchParams) {
  const { data, error } = await request<{ list: any[]; total: number; page: number; pageSize: number }>({
    url: '/user/getUserList',
    method: 'post',
    data: {
      page: params?.current ?? 1,
      pageSize: params?.size ?? 10,
      userName: params?.userName ?? '',
      nickName: params?.nickName ?? ''
    }
  });

  if (error || !data) {
    return { data: null as unknown as Api.SystemManage.UserList, error };
  }

  const records = (data.list || []).map((u: any) => ({
    id: u.ID ?? u.id,
    userName: u.userName,
    userGender: null,
    nickName: u.nickName,
    userPhone: u.phone ?? '',
    userEmail: u.email ?? '',
    userRoles: u.authority?.authorityName ? [u.authority.authorityName] : [],
    status: u.enable === 1 ? '1' : '2',
    createBy: '',
    createTime: u.CreatedAt ?? '',
    updateBy: '',
    updateTime: u.UpdatedAt ?? ''
  }));

  return {
    data: {
      records,
      current: data.page,
      size: data.pageSize,
      total: data.total
    } as Api.SystemManage.UserList,
    error: null
  };
}

/** 菜单列表 —— 对接 gin-vue-admin POST /menu/getBaseMenuTree（树），再拍平成表格行 */
export async function fetchGetMenuList() {
  const { data, error } = await request<{ menus: any[] }>({
    url: '/menu/getBaseMenuTree',
    method: 'post',
    data: {}
  });

  if (error || !data) {
    return { data: null as unknown as Api.SystemManage.MenuList, error };
  }

  const records: any[] = [];

  const walk = (nodes: any[], parentId: number) => {
    (nodes || []).forEach(node => {
      const id = node.ID ?? node.id;
      const meta = node.meta ?? {};

      records.push({
        id,
        parentId,
        // gin 用 menuLevel：0 目录、1 菜单、2 按钮；没有子节点的一级菜单按「菜单」展示
        menuType: node.menuLevel === 2 ? '2' : node.children?.length ? '1' : '2',
        menuName: meta.title ?? node.name,
        routeName: node.name,
        routePath: node.path,
        component: node.component ?? '',
        order: node.sort ?? 1,
        // 不传 i18nKey：gin 的菜单没有前端 i18n 词条，传了会显示成 route.xxx
        i18nKey: '',
        icon: meta.icon ?? '',
        iconType: '1',
        status: '1',
        keepAlive: Boolean(meta.keepAlive),
        constant: false,
        href: '',
        hideInMenu: Boolean(node.hidden),
        activeMenu: meta.activeName ?? '',
        multiTab: false
      });

      if (node.children?.length) walk(node.children, id);
    });
  };

  walk(data.menus, 0);

  return {
    data: {
      records,
      current: 1,
      size: records.length || 10,
      total: records.length
    } as unknown as Api.SystemManage.MenuList,
    error: null
  };
}

/** get all pages */
export function fetchGetAllPages() {
  return request<string[]>({
    url: '/systemManage/getAllPages',
    method: 'get'
  });
}

/** get menu tree */
export function fetchGetMenuTree() {
  return request<Api.SystemManage.MenuTree[]>({
    url: '/systemManage/getMenuTree',
    method: 'get'
  });
}

/** 新增用户 —— gin-vue-admin POST /user/admin_register */
export function fetchAddUser(data: { userName: string; nickName?: string; userPhone?: string; userEmail?: string }) {
  return request({
    url: '/user/admin_register',
    method: 'post',
    data: {
      username: data.userName,
      password: '123456',
      nickName: data.nickName,
      phone: data.userPhone,
      email: data.userEmail,
      authorityId: 888,
      headerImg: ''
    }
  });
}

/** 编辑用户 —— gin-vue-admin PUT /user/setUserInfo */
export function fetchUpdateUser(data: {
  id: string | number;
  nickName?: string;
  userPhone?: string;
  userEmail?: string;
}) {
  return request({
    url: '/user/setUserInfo',
    method: 'put',
    data: {
      ID: Number(data.id),
      nickName: data.nickName,
      phone: data.userPhone,
      email: data.userEmail
    }
  });
}

/** 删除用户 —— gin-vue-admin DELETE /user/deleteUser */
export function fetchDeleteUser(id: string | number) {
  return request({
    url: '/user/deleteUser',
    method: 'delete',
    data: { id: Number(id) }
  });
}

import { request } from '../request';

/** gin-vue-admin 登录返回结构 */
interface GvaLoginResult {
  token: string;
  user?: Record<string, any>;
}

/**
 * Login
 *
 * @param userName User name
 * @param password Password
 */
export async function fetchLogin(userName: string, password: string) {
  const { data, error } = await request<GvaLoginResult>({
    url: '/base/login',
    method: 'post',
    data: {
      username: userName,
      password,
      captcha: '',
      captchaId: ''
    }
  });

  if (error || !data?.token) {
    return { data: null as unknown as Api.Auth.LoginToken, error: error ?? new Error('登录失败') };
  }

  // soybean 期望 { token, refreshToken }；gin-vue-admin 没有 refreshToken
  return {
    data: { token: data.token, refreshToken: data.token } as Api.Auth.LoginToken,
    error: null
  };
}

/** Get user info */
export async function fetchGetUserInfo() {
  const { data, error } = await request<{ userInfo: Record<string, any> }>({ url: '/user/getUserInfo' });

  if (error || !data?.userInfo) {
    return { data: null as unknown as Api.Auth.UserInfo, error: error ?? new Error('获取用户信息失败') };
  }

  const user = data.userInfo;

  // 映射成 soybean 的 UserInfo；静态路由模式下 roles 含超管角色即可放行全部页面
  return {
    data: {
      userId: String(user.id ?? user.ID ?? ''),
      userName: user.userName ?? user.username ?? '',
      roles: ['R_SUPER'],
      buttons: []
    } as Api.Auth.UserInfo,
    error: null
  };
}

/**
 * Refresh token
 *
 * gin-vue-admin 没有独立的 refresh 接口，这里用原 token 续期（保持接口形状）
 *
 * @param refreshToken Refresh token
 */
export function fetchRefreshToken(refreshToken: string) {
  return request<Api.Auth.LoginToken>({
    url: '/base/login',
    method: 'post',
    data: { token: refreshToken }
  });
}

/**
 * return custom backend error
 *
 * @param code error code
 * @param msg error message
 */
export function fetchCustomBackendError(code: string, msg: string) {
  return request({ url: '/auth/error', params: { code, msg } });
}

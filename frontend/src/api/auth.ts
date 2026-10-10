import {http} from './client'
import type {LoginResult} from './types'

export function login(username: string, password: string, deviceName: string) {
  return http.post<LoginResult>('/auth/login', {username, password, device_name: deviceName})
}

/**
 * 登出：吊销令牌对应的设备，服务端同时清除浏览器中的文件 Cookie。
 * token 省略时登出当前账号；退出其他账号时传入该账号的令牌。
 */
export function logout(token?: string) {
  return http.post('/auth/logout', {}, {token})
}

/**
 * 为令牌对应的设备签发新的文件令牌，服务端写入文件 Cookie（HttpOnly，只发往 /api/file）。
 * 页面中的图片、音视频、PDF 和下载链接由浏览器直接请求，靠这个 Cookie 认证。token 省略时使用当前账号。
 */
export function writeFileCookie(token?: string) {
  return http.post('/auth/file_cookie', {}, {token})
}

/** 注销令牌对应设备的文件令牌并清除文件 Cookie，设备保持登录。切换到其他账号前调用。 */
export function revokeFileCookie(token: string) {
  return http.post('/auth/file_cookie/revoke', {}, {token})
}

import {http} from './client'
import type {LoginResult} from './types'

export function login(username: string, password: string, deviceName: string) {
  return http.post<LoginResult>('/auth/login', {username, password, device_name: deviceName})
}

/** 登出，服务端同时清除文件 Cookie。 */
export function logout() {
  return http.post('/auth/logout')
}

/**
 * 让服务端把当前令牌写入文件 Cookie（HttpOnly，只发往 /api/file）。
 * 页面中的图片、音视频、PDF 和下载链接由浏览器直接请求，靠这个 Cookie 认证。
 */
export function writeFileCookie() {
  return http.post('/auth/file_cookie')
}

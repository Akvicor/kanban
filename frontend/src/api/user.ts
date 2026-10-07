import {http} from './client'
import type {Account, Me, Settings, SettingsChanges} from './types'

export function fetchMe() {
  return http.get<Me>('/user/me')
}

/** 修改自己的用户名和昵称，返回修改后的账号和同步序号。用户名仍用于登录。 */
export function updateProfile(username: string, nickname: string) {
  return http.write<Account>('/user/profile/update', {username, nickname})
}

/** 保存个人设置，返回保存后的完整设置和同步序号。 */
export function updateSettings(changes: SettingsChanges) {
  return http.write<Settings>('/user/settings/update', changes)
}

/** 修改密码。其他设备的吊销通过同步推送到达。 */
export function changePassword(currentPassword: string, newPassword: string) {
  return http.post('/user/password/update', {current_password: currentPassword, new_password: newPassword})
}

export function revokeDevice(id: number) {
  return http.write('/user/device/revoke', {id})
}

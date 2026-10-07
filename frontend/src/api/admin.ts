import {http} from './client'
import type {AdminUser} from './types'

export function listUsers() {
  return http.get<AdminUser[]>('/admin/user/list')
}

export function createUser(username: string, password: string) {
  return http.post<AdminUser>('/admin/user/create', {username, password})
}

export function disableUser(id: number) {
  return http.post('/admin/user/disable', {id})
}

export function enableUser(id: number) {
  return http.post('/admin/user/enable', {id})
}

export function resetPassword(id: number, password: string) {
  return http.post('/admin/user/reset_password', {id, password})
}

export function deleteUser(id: number) {
  return http.post('/admin/user/delete', {id})
}

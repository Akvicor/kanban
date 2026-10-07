import {http} from './client'
import type {Board, Folder, Settings} from './types'

/**
 * 看板目录的写入接口。parentId / folderId 为 null 表示根；index 是在同级中的位置（从 0 开始），为 null 时排在末尾。
 * 写入后返回受影响的文件夹或看板和同步序号，其余变化通过同步推送到达。
 */

export function createFolder(parentId: number | null, name: string, index: number | null = null) {
  return http.write<Folder>('/folder/create', {parent_id: parentId, name, index})
}

export function renameFolder(id: number, name: string) {
  return http.write<Folder>('/folder/rename', {id, name})
}

export function moveFolder(id: number, parentId: number | null, index: number | null) {
  return http.write<Folder>('/folder/move', {id, parent_id: parentId, index})
}

export function archiveFolder(id: number) {
  return http.write('/folder/archive', {id})
}

export function createBoard(folderId: number | null, name: string, index: number | null = null) {
  return http.write<Board>('/board/create', {folder_id: folderId, name, index})
}

export function renameBoard(id: number, name: string) {
  return http.write<Board>('/board/rename', {id, name})
}

export function moveBoard(id: number, folderId: number | null, index: number | null) {
  return http.write<Board>('/board/move', {id, folder_id: folderId, index})
}

export function archiveBoard(id: number) {
  return http.write('/board/archive', {id})
}

export function restoreBoard(id: number, folderId: number | null, index: number | null) {
  return http.write<Board>('/board/restore', {id, folder_id: folderId, index})
}

/** 设置主看板，id 为 null 时取消。返回修改后的个人设置。 */
export function setMainBoard(id: number | null) {
  return http.write<Settings>('/board/main', {id})
}

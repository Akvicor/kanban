import {http} from './client'
import type {Attachment, Card, UploadState, UserFileList} from './types'

/**
 * 上传、附件、封面和文件管理的接口，以及文件内容和缩略图的地址。
 * 文件地址由浏览器直接请求（图片、音视频、PDF、下载），靠文件 Cookie 认证，见 auth.writeFileCookie。
 */

/** 缩略图规格：短边像素。 */
export type ThumbnailSize = 360 | 720

/** 开始或继续上传：系统已有时返回 exists，否则返回上传会话和服务端已收到的字节数。 */
export function prepareUpload(sha256: string, size: number) {
  return http.post<UploadState>('/upload/prepare', {sha256, size})
}

/** 上传一个分片，offset 必须等于服务端已收到的字节数。收齐并校验通过时返回 done。 */
export function uploadChunk(sessionId: number, offset: number, chunk: Blob) {
  return http.raw<UploadState>(`/upload/chunk?session_id=${sessionId}&offset=${offset}`, chunk)
}

/** 用已上传的文件（按 sha256）新建文件附件。 */
export function createFileAttachment(cardId: number, sha256: string, name: string) {
  return http.write<Attachment>('/attachment/create_file', {card_id: cardId, sha256, name})
}

/** 新建链接附件，name 为空时取网址。站点图标在后台抓取，抓到后通过推送更新。 */
export function createLinkAttachment(cardId: number, url: string, name: string) {
  return http.write<Attachment>('/attachment/create_link', {card_id: cardId, url, name})
}

export function renameAttachment(id: number, name: string) {
  return http.write<Attachment>('/attachment/rename', {id, name})
}

export function deleteAttachment(id: number) {
  return http.write('/attachment/delete', {id})
}

/** 设置卡片封面，attachmentId 为 null 时取消。 */
export function setCardCover(cardId: number, attachmentId: number | null) {
  return http.write<Card>('/card/cover', {id: cardId, attachment_id: attachmentId})
}

/** 用户的全部文件，用于文件管理。 */
export function fetchUserFiles() {
  return http.get<UserFileList>('/user_file/list')
}

/** 删除引用数为 0 的文件。 */
export function deleteUserFile(id: number) {
  return http.write('/user_file/delete', {id})
}

/** 文件附件的内容地址，下载时文件名为附件名称。 */
export function attachmentUrl(id: number): string {
  return `/api/file/attachment/${id}`
}

export function attachmentThumbnailUrl(id: number, size: ThumbnailSize): string {
  return `/api/file/attachment/${id}/thumbnail/${size}`
}

/** 文件管理中某个文件的内容地址。 */
export function userFileUrl(id: number): string {
  return `/api/file/user_file/${id}`
}

export function userFileThumbnailUrl(id: number, size: ThumbnailSize): string {
  return `/api/file/user_file/${id}/thumbnail/${size}`
}

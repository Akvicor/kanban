import {describe, expect, it} from 'vitest'
import type {Attachment, Card, FileInfo, UserFile} from '../api/types'
import {sortFiles} from '../pages/files/sortFiles'
import {previewKind} from './kind'
import {attachmentSummaries} from './summary'

const info = (mime: string, image = false): FileInfo => ({size: 10, mime_type: mime, image, width: 0, height: 0})

describe('previewKind', () => {
  it('图片按是否生成缩略图，PDF 和音视频按服务端识别的类型，文本类按扩展名', () => {
    expect(previewKind(info('image/png', true), 'a.png')).toBe('image')
    expect(previewKind(info('image/heic'), 'a.heic')).toBe('none')
    expect(previewKind(info('application/pdf'), 'a.bin')).toBe('pdf')
    expect(previewKind(info('video/mp4'), 'a.mp4')).toBe('video')
    expect(previewKind(info('audio/mpeg'), 'a.mp3')).toBe('audio')
    expect(previewKind(info('text/plain'), 'data.CSV')).toBe('csv')
    expect(previewKind(info('text/plain'), 'README.md')).toBe('markdown')
    expect(previewKind(info('application/octet-stream'), 'main.go')).toBe('text')
    expect(previewKind(info('text/plain'), 'notes')).toBe('text')
    expect(previewKind(info('application/zip'), 'a.zip')).toBe('none')
  })
})

function card(id: number, cover: number | null): Card {
  return {
    id, panel_id: 1, list_id: 1, position: 1, title: '', description: '', label_ids: [], priority_level_id: null,
    remind_at: null, due_at: null, remind_notify: false, due_notify: false, created_at: '', started_at: null,
    completed_at: null, timer_seconds: 0, timer_started_at: null, archived_at: null, cover_attachment_id: cover, notify_channel_ids: [],
  }
}

function attachment(id: number, cardId: number, image: boolean): Attachment {
  return {id, card_id: cardId, type: 'file', name: `${id}`, file: {...info('image/png', image), user_file_id: id}, url: '', favicon: '', created_at: ''}
}

describe('attachmentSummaries', () => {
  it('统计附件数，封面只在附件存在且是图片时显示', () => {
    const summaries = attachmentSummaries(
      [card(1, 11), card(2, 21), card(3, 99), card(4, null)],
      [attachment(11, 1, true), attachment(12, 1, false), attachment(21, 2, false)],
    )
    expect(summaries.get(1)).toEqual({count: 2, coverUrl: '/api/file/attachment/11/thumbnail/360'})
    expect(summaries.get(2)).toEqual({count: 1, coverUrl: null})
    expect(summaries.has(3)).toBe(false)
    expect(summaries.has(4)).toBe(false)
  })
})

describe('sortFiles', () => {
  const file = (id: number, first: string, zero: string | null): UserFile => ({
    ...info('text/plain'), id, name: `${id}`, reference_count: zero ? 0 : 1, zero_at: zero, first_referenced_at: first,
  })
  const files = [file(1, '2026-01-03', null), file(2, '2026-01-01', '2026-02-02'), file(3, '2026-01-02', '2026-02-01'), file(4, '2026-01-04', null)]

  it('默认未被引用的在前，按归零时间从早到晚；其余按首次引用时间从近到远', () => {
    expect(sortFiles(files, 'unused').map((f) => f.id)).toEqual([3, 2, 4, 1])
  })
  it('可以按首次引用时间从近到远排列', () => {
    expect(sortFiles(files, 'first').map((f) => f.id)).toEqual([4, 1, 3, 2])
  })
})

import {attachmentThumbnailUrl} from '../api/file'
import type {Attachment, Card} from '../api/types'

/** 卡片正面显示的附件信息：附件数和封面缩略图地址（没有封面时为 null）。 */
export interface AttachmentSummary {
  count: number
  coverUrl: string | null
}

export const EMPTY_SUMMARY: AttachmentSummary = {count: 0, coverUrl: null}

/**
 * 按卡片汇总附件。封面只在封面附件已加载、并且是图片时显示（例如删除推送先于卡片更新到达时不显示失效的封面）。
 */
export function attachmentSummaries(cards: Card[], attachments: Attachment[]): Map<number, AttachmentSummary> {
  const byId = new Map(attachments.map((attachment) => [attachment.id, attachment]))
  const counts = new Map<number, number>()
  for (const attachment of attachments) {
    counts.set(attachment.card_id, (counts.get(attachment.card_id) ?? 0) + 1)
  }
  const result = new Map<number, AttachmentSummary>()
  for (const card of cards) {
    const cover = card.cover_attachment_id === null ? undefined : byId.get(card.cover_attachment_id)
    const coverUrl = cover && cover.card_id === card.id && cover.file?.image ? attachmentThumbnailUrl(cover.id, 360) : null
    const count = counts.get(card.id) ?? 0
    if (count > 0 || coverUrl) {
      result.set(card.id, {count, coverUrl})
    }
  }
  return result
}

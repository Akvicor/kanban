import type {Card, List, NotifyDelivery, NotifyKind} from '../api/types'

/**
 * 卡片的一种通知在一条渠道上的发送状态：
 * - future：时间未到；
 * - waiting：时间已到，等待后台任务发送；
 * - sent：已发送；
 * - failed：发送失败，等待重试。
 */
export type DeliveryStatus =
  | {state: 'future'}
  | {state: 'waiting'}
  | {state: 'sent'; sentAt: string}
  | {state: 'failed'; nextAttemptAt: string; error: string}

/** 计算发送状态所需的卡片位置：卡片、所在列表，以及所在面板或看板是否已归档。 */
export interface NotifyPlace {
  card: Card
  list: List | undefined
  panelArchived: boolean
}

/**
 * 计算发送状态，条件与后端后台发送任务一致。卡片当前不满足发送条件时返回 null：没有设置这个时间、
 * 关闭了通知开关、所在列表关闭了这种通知、卡片不在列表中，或卡片、列表、面板、看板已归档。
 * 只认针对当前时间值的发送记录。
 */
export function deliveryStatus(
  {card, list, panelArchived}: NotifyPlace,
  kind: NotifyKind,
  channelId: number,
  deliveries: NotifyDelivery[],
  now: number,
): DeliveryStatus | null {
  const at = kind === 'remind' ? card.remind_at : card.due_at
  const notify = kind === 'remind' ? card.remind_notify : card.due_notify
  if (!at || !notify || !list || card.archived_at !== null || list.archived_at !== null || panelArchived) return null
  if (kind === 'remind' ? list.remind_off : list.due_off) return null
  const target = Date.parse(at)
  const record = deliveries.find(
    (delivery) => delivery.card_id === card.id && delivery.kind === kind && delivery.channel_id === channelId && Date.parse(delivery.target_at) === target,
  )
  if (record?.sent_at) return {state: 'sent', sentAt: record.sent_at}
  if (record && record.attempts > 0) return {state: 'failed', nextAttemptAt: record.next_attempt_at, error: record.last_error}
  return target > now ? {state: 'future'} : {state: 'waiting'}
}

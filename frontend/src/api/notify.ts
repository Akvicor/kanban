import {http} from './client'
import type {Card, NotifyChannel, NotifyFormat} from './types'

/** 新建或修改通知渠道的内容。修改时 token、sign 为空字符串表示保留原值。 */
export interface ChannelInput {
  name: string
  api: string
  token: string
  sign: string
  format: NotifyFormat
}

export function createChannel(input: ChannelInput) {
  return http.write<NotifyChannel>('/notify_channel/create', input)
}

export function updateChannel(id: number, input: ChannelInput) {
  return http.write<NotifyChannel>('/notify_channel/update', {id, ...input})
}

export function deleteChannel(id: number) {
  return http.write('/notify_channel/delete', {id})
}

/** 用渠道发送一条测试消息，失败时抛出带错误信息的异常。 */
export function testChannel(id: number) {
  return http.post('/notify_channel/test', {id})
}

/** 为卡片选择（on 为 true）或取消选择通知渠道。 */
export function setCardChannel(cardId: number, channelId: number, on: boolean) {
  return http.write<Card>('/card/notify_channel', {id: cardId, channel_id: channelId, on})
}

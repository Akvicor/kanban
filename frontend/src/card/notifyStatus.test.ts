import {describe, expect, it} from 'vitest'
import type {Card, List, NotifyDelivery} from '../api/types'
import {deliveryStatus, type NotifyPlace} from './notifyStatus'

const now = Date.parse('2026-03-01T10:00:00Z')

function place(cardChanges: Partial<Card> = {}, listChanges: Partial<List> = {}, panelArchived = false): NotifyPlace {
  const card = {
    id: 1, list_id: 2, remind_at: '2026-03-01T09:00:00Z', due_at: '2026-03-02T09:00:00Z', remind_notify: true, due_notify: true,
    archived_at: null, ...cardChanges,
  } as Card
  const list = {id: 2, remind_off: false, due_off: false, archived_at: null, ...listChanges} as List
  return {card, list, panelArchived}
}

function delivery(changes: Partial<NotifyDelivery>): NotifyDelivery {
  return {
    id: 9, card_id: 1, channel_id: 5, kind: 'remind', target_at: '2026-03-01T09:00:00Z', sent_at: null, attempts: 0,
    next_attempt_at: '2026-03-01T09:00:00Z', last_error: '', ...changes,
  }
}

describe('deliveryStatus', () => {
  it('按时间是否已到和发送记录推导状态', () => {
    expect(deliveryStatus(place(), 'due', 5, [], now)).toEqual({state: 'future'})
    expect(deliveryStatus(place(), 'remind', 5, [], now)).toEqual({state: 'waiting'})
    expect(deliveryStatus(place(), 'remind', 5, [delivery({sent_at: '2026-03-01T09:00:05Z', attempts: 1})], now)).toEqual({
      state: 'sent',
      sentAt: '2026-03-01T09:00:05Z',
    })
    expect(
      deliveryStatus(place(), 'remind', 5, [delivery({attempts: 2, next_attempt_at: '2026-03-01T10:02:00Z', last_error: '超时'})], now),
    ).toEqual({state: 'failed', nextAttemptAt: '2026-03-01T10:02:00Z', error: '超时'})
  })

  it('只认针对当前时间值和这条渠道的记录', () => {
    const old = delivery({target_at: '2026-02-28T09:00:00Z', sent_at: '2026-02-28T09:00:05Z', attempts: 1})
    const other = delivery({channel_id: 6, sent_at: '2026-03-01T09:00:05Z', attempts: 1})
    expect(deliveryStatus(place(), 'remind', 5, [old, other], now)).toEqual({state: 'waiting'})
  })

  it('不满足发送条件时没有状态', () => {
    expect(deliveryStatus(place({remind_at: null}), 'remind', 5, [], now)).toBeNull()
    expect(deliveryStatus(place({remind_notify: false}), 'remind', 5, [], now)).toBeNull()
    expect(deliveryStatus(place({}, {remind_off: true}), 'remind', 5, [], now)).toBeNull()
    expect(deliveryStatus(place({}, {remind_off: true}), 'due', 5, [], now)).toEqual({state: 'future'})
    expect(deliveryStatus(place({archived_at: '2026-03-01T00:00:00Z'}), 'remind', 5, [], now)).toBeNull()
    expect(deliveryStatus(place({}, {archived_at: '2026-03-01T00:00:00Z'}), 'remind', 5, [], now)).toBeNull()
    expect(deliveryStatus(place({}, {}, true), 'remind', 5, [], now)).toBeNull()
    expect(deliveryStatus({...place(), list: undefined}, 'remind', 5, [], now)).toBeNull()
  })
})

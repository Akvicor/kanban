import {fireEvent, render, screen} from '@testing-library/react'
import {afterEach, describe, expect, it, vi} from 'vitest'
import type {Card} from '../api/types'
import {CardItem, type CardFaceContext} from './CardItem'

const card = {
  id: 7,
  panel_id: 10,
  list_id: 5,
  position: 100,
  title: '整理预算',
  description: '',
  label_ids: [],
  priority_level_id: null,
  remind_at: null,
  due_at: null,
  remind_notify: false,
  due_notify: false,
  created_at: '2026-01-01T00:00:00Z',
  started_at: null,
  completed_at: null,
  timer_seconds: 0,
  timer_started_at: null,
  archived_at: null,
  cover_attachment_id: null,
  notify_channel_ids: [],
} as Card

const context: CardFaceContext = {
  labels: new Map(),
  levels: new Map(),
  timeZone: 'UTC',
  now: Date.parse('2026-01-01T00:00:00Z'),
  showAge: false,
  progress: {done: 0, total: 0},
  attachments: {count: 0, coverUrl: null},
}

/** 两次点击间隔 threshold 毫秒。 */
function clickTwice(element: HTMLElement, gapMs: number) {
  fireEvent.click(element)
  vi.advanceTimersByTime(gapMs)
  fireEvent.click(element)
}

describe('CardItem', () => {
  afterEach(() => vi.useRealTimers())

  it('连点两下打开卡片详情，间隔过久的两次点击不打开', () => {
    vi.useFakeTimers()
    const onOpen = vi.fn()
    const {container} = render(<CardItem card={card} context={context} dimmed={false} selected={false} moving={false} dropHint={null} onOpen={onOpen} onHover={vi.fn()} />)
    const face = container.querySelector('.card') as HTMLElement

    clickTwice(face, 100)
    expect(onOpen).toHaveBeenCalledTimes(1)

    clickTwice(face, 600)
    expect(onOpen).toHaveBeenCalledTimes(1)
  })

  it('点菜单按钮不算双击，点完菜单再点卡片两下仍然打开', () => {
    vi.useFakeTimers()
    const onOpen = vi.fn()
    const {container} = render(
      <CardItem
        card={card}
        context={context}
        dimmed={false}
        selected={false}
        moving={false}
        dropHint={null}
        onOpen={onOpen}
        onHover={vi.fn()}
        onCopy={vi.fn()}
        onCut={vi.fn()}
        onEdit={vi.fn()}
      />,
    )
    const face = container.querySelector('.card') as HTMLElement

    fireEvent.click(screen.getByRole('button', {name: '整理预算 的菜单'}))
    fireEvent.click(face)
    expect(onOpen).not.toHaveBeenCalled()
    fireEvent.click(face)
    expect(onOpen).toHaveBeenCalledTimes(1)
  })
})

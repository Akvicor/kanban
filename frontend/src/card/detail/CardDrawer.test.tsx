import {act, render, screen} from '@testing-library/react'
import {afterEach, describe, expect, it, vi} from 'vitest'
import type {Panel} from '../../api/types'
import {CardDrawer} from './CardDrawer'

// 只验证抽屉外框的开合和内容保留，卡片详情本身用占位元素代替。
vi.mock('./CardDetail', () => ({
  CardDetail: ({cardId}: {cardId: number}) => <p>卡片 {cardId}</p>,
}))

const panel: Panel = {id: 10, board_id: 1, name: '默认', position: 1, archived_at: null}

function drawer(cardId: number | null, openKey: string) {
  return <CardDrawer panel={panel} cardId={cardId} openKey={openKey} focus={null} onClose={vi.fn()} onNotice={vi.fn()} />
}

const frame = () => document.querySelector('.drawer-frame') as HTMLElement

describe('CardDrawer', () => {
  afterEach(() => vi.useRealTimers())

  it('切换卡片时保持打开；关闭后滑出期间仍显示刚才的卡片，滑出后移除', () => {
    vi.useFakeTimers()
    const {rerender} = render(drawer(null, 'a'))
    expect(frame()).not.toHaveClass('open')
    expect(frame()).toBeEmptyDOMElement()

    rerender(drawer(7, 'b'))
    expect(frame()).toHaveClass('open')
    expect(screen.getByText('卡片 7')).toBeInTheDocument()

    rerender(drawer(8, 'c'))
    expect(frame()).toHaveClass('open')
    expect(screen.getByText('卡片 8')).toBeInTheDocument()

    rerender(drawer(null, 'd'))
    expect(frame()).not.toHaveClass('open')
    expect(screen.getByText('卡片 8')).toBeInTheDocument()
    act(() => vi.advanceTimersByTime(250))
    expect(screen.queryByText('卡片 8')).toBeNull()
  })
})

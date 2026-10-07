import {act, renderHook} from '@testing-library/react'
import {describe, expect, it} from 'vitest'
import {usePendingMoves} from './pendingMoves'

/** 可以从外部决定结果的请求。 */
function deferred() {
  let resolve!: () => void
  let reject!: (err: Error) => void
  const promise = new Promise<void>((res, rej) => {
    resolve = res
    reject = rej
  })
  return {promise, resolve, reject}
}

describe('usePendingMoves', () => {
  it('位置未变时保持移动中，位置一变立即结束，不必等请求完成', () => {
    const {result} = renderHook(() => usePendingMoves())
    const request = deferred()
    act(() => result.current.track('c:1', '10:100', request.promise))

    expect(result.current.isPending('c:1', '10:100')).toBe(true)
    // 同步推送先到，卡片已在新位置。
    expect(result.current.isPending('c:1', '20:50')).toBe(false)
    expect(result.current.isPending('c:2', '10:100')).toBe(false)
  })

  it('请求失败时清除记录，对象恢复原样', async () => {
    const {result} = renderHook(() => usePendingMoves())
    const request = deferred()
    act(() => result.current.track('l:3', '200', request.promise))
    expect(result.current.isPending('l:3', '200')).toBe(true)

    await act(async () => {
      request.reject(new Error('network'))
      await request.promise.catch(() => undefined)
    })
    expect(result.current.isPending('l:3', '200')).toBe(false)
  })

  it('同一对象再次拖动后，前一次请求结束不会清掉新的记录', async () => {
    const {result} = renderHook(() => usePendingMoves())
    const first = deferred()
    const second = deferred()
    act(() => result.current.track('c:1', '10:100', first.promise))
    act(() => result.current.track('c:1', '20:50', second.promise))

    await act(async () => {
      first.resolve()
      await first.promise
    })
    expect(result.current.isPending('c:1', '20:50')).toBe(true)

    await act(async () => {
      second.resolve()
      await second.promise
    })
    expect(result.current.isPending('c:1', '20:50')).toBe(false)
  })
})

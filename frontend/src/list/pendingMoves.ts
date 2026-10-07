/**
 * 拖放落点后等待生效的移动。
 *
 * 松手时记下被拖对象原来的位置签名（卡片是所在列和序号，列表是序号）。对象当前的签名仍等于记录时，
 * 它就处在「移动中」：界面保持拖动时的半透明，让用户看得出移动还没完成。
 * 新位置可能由同步推送或写入响应带来，二者先后不定；只要签名一变，半透明就随布局变化、落点动画同帧结束。
 * 请求结束（成功或失败）时清除记录：成功时签名早已变化；失败时位置未变，清除后恢复原样。
 */
import {useCallback, useState} from 'react'

export interface PendingMoves {
  /** 登记一次已发出的移动请求。key 标识被拖对象，from 是它松手前的位置签名。 */
  track: (key: string, from: string, request: Promise<unknown>) => void
  /** 对象是否仍在等待移动生效：有登记且当前位置签名与松手前相同。 */
  isPending: (key: string, current: string) => boolean
}

export function usePendingMoves(): PendingMoves {
  const [pending, setPending] = useState<ReadonlyMap<string, string>>(() => new Map())

  const track = useCallback((key: string, from: string, request: Promise<unknown>) => {
    setPending((prev) => new Map(prev).set(key, from))
    // 同一对象在请求结束前又被拖动时，记录已换成新的 from，这里只清除自己登记的那一条。
    const done = () =>
      setPending((prev) => {
        if (prev.get(key) !== from) return prev
        const next = new Map(prev)
        next.delete(key)
        return next
      })
    // 失败由调用方提示，这里只在请求结束时清除记录。
    request.then(done, done)
  }, [])

  const isPending = useCallback((key: string, current: string) => pending.get(key) === current, [pending])

  return {track, isPending}
}

import {useEffect, useState} from 'react'

/** 当前时刻（毫秒），每隔 intervalMs 更新一次。用于卡龄、到期状态和定时器读数随时间变化。 */
export function useNow(intervalMs: number): number {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), intervalMs)
    return () => clearInterval(timer)
  }, [intervalMs])
  return now
}

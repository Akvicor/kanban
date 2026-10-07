import {useCallback, useEffect, useRef, useState} from 'react'
import './Toast.css'

/** 短暂显示一条提示的时长。 */
const TOAST_MS = 3000

/** 返回显示提示的函数和提示元素。新的提示会替换正在显示的提示。 */
export function useToast() {
  const [message, setMessage] = useState('')
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const show = useCallback((text: string) => {
    if (timer.current) clearTimeout(timer.current)
    setMessage(text)
    timer.current = setTimeout(() => setMessage(''), TOAST_MS)
  }, [])

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current)
  }, [])

  const element = (
    <div className={message ? 'toast show' : 'toast'} role="status">
      {message}
    </div>
  )
  return {show, element}
}

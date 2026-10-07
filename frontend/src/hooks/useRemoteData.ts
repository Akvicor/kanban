import {useCallback, useEffect, useState} from 'react'
import {errorMessage} from '../api/client'

/**
 * 加载一份服务端数据，返回数据、加载错误和重新加载函数。
 * load 需要是稳定的函数引用（例如 api 模块中的函数），否则每次渲染都会重新加载。
 */
export function useRemoteData<T>(load: () => Promise<T>, initial: T) {
  const [state, setState] = useState({data: initial, error: ''})
  const [version, setVersion] = useState(0)

  useEffect(() => {
    let cancelled = false
    load()
      .then((data) => {
        if (!cancelled) setState({data, error: ''})
      })
      .catch((err: unknown) => {
        if (!cancelled) setState((current) => ({...current, error: errorMessage(err)}))
      })
    return () => {
      cancelled = true
    }
  }, [load, version])

  const reload = useCallback(() => setVersion((current) => current + 1), [])
  return {data: state.data, error: state.error, reload}
}

import {useCallback, useState} from 'react'

/** 本设备保存目录中已展开文件夹的键。展开状态是界面状态，不同步到其他设备。 */
const STORAGE_KEY = 'kanban.directory.expanded'

function read(): Set<number> {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '[]')
    return new Set(Array.isArray(value) ? value.filter((id): id is number => typeof id === 'number') : [])
  } catch {
    return new Set()
  }
}

/** 目录中已展开的文件夹，保存在本设备。 */
export function useExpandedFolders() {
  const [expanded, setExpanded] = useState(read)

  const setOpen = useCallback((id: number, open: boolean) => {
    setExpanded((current) => {
      const next = new Set(current)
      if (open) next.add(id)
      else next.delete(id)
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify([...next]))
      } catch {
        // 存储不可用时只在本次打开期间记住。
      }
      return next
    })
  }, [])

  return {expanded, setOpen}
}

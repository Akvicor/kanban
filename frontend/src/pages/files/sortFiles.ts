import type {UserFile} from '../../api/types'

/** 文件管理的排序方式。 */
export type FileSort = 'unused' | 'first'

/**
 * 文件排序：
 * - unused（默认）：未被引用的在前，按归零时间从早到晚，便于找出长期不用的文件；被引用的在后，按首次引用时间从近到远。
 * - first：按首次引用时间从近到远。
 */
export function sortFiles(files: UserFile[], sort: FileSort): UserFile[] {
  const byFirst = (a: UserFile, b: UserFile) => b.first_referenced_at.localeCompare(a.first_referenced_at) || b.id - a.id
  if (sort === 'first') return [...files].sort(byFirst)
  return [...files].sort((a, b) => {
    if (a.zero_at !== null && b.zero_at !== null) return a.zero_at.localeCompare(b.zero_at) || a.id - b.id
    if (a.zero_at !== null) return -1
    if (b.zero_at !== null) return 1
    return byFirst(a, b)
  })
}

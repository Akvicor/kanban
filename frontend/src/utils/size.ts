const UNITS = ['B', 'KB', 'MB', 'GB', 'TB']

/** 文件大小的显示文字，按 1024 进位，例如 512 B、1.5 MB。 */
export function formatSize(bytes: number): string {
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024
    unit++
  }
  const text = unit === 0 || value >= 100 ? Math.round(value).toString() : value.toFixed(1).replace(/\.0$/, '')
  return `${text} ${UNITS[unit]}`
}

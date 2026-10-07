/** 日期时间显示。界面统一使用 YYYY-MM-DD HH:mm，按用户在个人设置中的时区换算。 */

const formatters = new Map<string, Intl.DateTimeFormat>()

function formatter(timeZone: string): Intl.DateTimeFormat {
  let cached = formatters.get(timeZone)
  if (!cached) {
    cached = new Intl.DateTimeFormat('en-CA', {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      hourCycle: 'h23',
    })
    formatters.set(timeZone, cached)
  }
  return cached
}

/** 把时刻格式化为 YYYY-MM-DD HH:mm。 */
export function formatDateTime(value: string | Date, timeZone: string): string {
  const parts = Object.fromEntries(
    formatter(timeZone)
      .formatToParts(typeof value === 'string' ? new Date(value) : value)
      .map((part) => [part.type, part.value]),
  )
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}`
}

/** 时区在某一时刻相对 UTC 的偏移（毫秒）。 */
function offsetAt(instant: number, timeZone: string): number {
  const parts = Object.fromEntries(
    new Intl.DateTimeFormat('en-US', {
      timeZone,
      hourCycle: 'h23',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
      .formatToParts(instant)
      .map((part) => [part.type, Number(part.value)]),
  )
  const asUTC = Date.UTC(parts.year, parts.month - 1, parts.day, parts.hour, parts.minute, parts.second)
  return asUTC - Math.floor(instant / 1000) * 1000
}

/** 把时刻转换为用户时区中的 datetime-local 输入值（YYYY-MM-DDTHH:mm），值为空时返回空字符串。 */
export function toZonedInput(value: string | null, timeZone: string): string {
  if (!value) return ''
  return formatDateTime(value, timeZone).replace(' ', 'T')
}

/**
 * 把 datetime-local 输入值按用户时区解释为时刻，返回 RFC 3339 字符串；输入为空或无效时返回 null。
 * 夏令时切换时不存在的时间按切换后的偏移计算。
 */
export function fromZonedInput(value: string, timeZone: string): string | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})$/.exec(value)
  if (!match) return null
  const [, y, mo, d, h, mi] = match.map(Number)
  const wall = Date.UTC(y, mo - 1, d, h, mi)
  // 先按墙上时间所在时刻的偏移换算，再用换算结果的偏移修正一次。
  let instant = wall - offsetAt(wall, timeZone)
  instant = wall - offsetAt(instant, timeZone)
  return new Date(instant).toISOString()
}

/** 浏览器支持的 IANA 时区列表。current 不在列表中时也加入，保证当前设置可以显示。 */
export function timeZoneOptions(current: string): string[] {
  const zones = new Set(Intl.supportedValuesOf('timeZone'))
  zones.add('UTC')
  zones.add(current)
  return [...zones].sort()
}

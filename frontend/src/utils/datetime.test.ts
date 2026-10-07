import {describe, expect, it} from 'vitest'
import {formatDateTime, fromZonedInput, timeZoneOptions, toZonedInput} from './datetime'

describe('formatDateTime', () => {
  it('按指定时区格式化为 YYYY-MM-DD HH:mm', () => {
    const instant = '2026-10-02T16:05:00Z'
    expect(formatDateTime(instant, 'UTC')).toBe('2026-10-02 16:05')
    expect(formatDateTime(instant, 'Asia/Shanghai')).toBe('2026-10-03 00:05')
    expect(formatDateTime(instant, 'America/New_York')).toBe('2026-10-02 12:05')
  })
})

describe('toZonedInput / fromZonedInput', () => {
  it('按用户时区换算，与浏览器所在时区无关', () => {
    expect(toZonedInput('2026-10-02T16:05:00Z', 'Asia/Shanghai')).toBe('2026-10-03T00:05')
    expect(fromZonedInput('2026-10-03T00:05', 'Asia/Shanghai')).toBe('2026-10-02T16:05:00.000Z')
    expect(fromZonedInput('2026-07-01T09:00', 'America/New_York')).toBe('2026-07-01T13:00:00.000Z')
    expect(fromZonedInput('2026-01-01T09:00', 'America/New_York')).toBe('2026-01-01T14:00:00.000Z')
  })
  it('空值和无效输入', () => {
    expect(toZonedInput(null, 'UTC')).toBe('')
    expect(fromZonedInput('', 'UTC')).toBeNull()
  })
})

describe('timeZoneOptions', () => {
  it('包含当前设置的时区', () => {
    const options = timeZoneOptions('Etc/GMT-8')
    expect(options).toContain('Etc/GMT-8')
    expect(options).toContain('UTC')
  })
})

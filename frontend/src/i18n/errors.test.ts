/**
 * 错误码与文案的完整性比对：后端 resp 包定义的错误码与前端双语映射必须一一对应。
 * 后端新增错误码而前端缺少任一语言的文案时，此测试失败，防止修改不全。
 */
import {readFileSync} from 'node:fs'
import {resolve} from 'node:path'
import {describe, expect, it} from 'vitest'
import {errorMessages as errorsEn} from './errors.en'
import {errorMessages as errorsZhCN} from './errors.zh-CN'
import {messages as messagesEn} from './messages.en'
import {messages as messagesZhCN} from './messages.zh-CN'

// vitest 的工作目录是 frontend，后端错误码源码在仓库的 backend 下。
const RESP_DIR = resolve(process.cwd(), '../backend/cmd/app/server/common/resp')

/** 从 Go 源码提取 `Name Code = 数字` 常量。 */
function parseCodes(file: string): Map<string, number> {
  const codes = new Map<string, number>()
  for (const match of readFileSync(resolve(RESP_DIR, file), 'utf8').matchAll(/^\t(\w+)\s+Code = (\d+)/gm)) {
    codes.set(match[1], Number(match[2]))
  }
  return codes
}

describe('错误码文案', () => {
  const backendCodes = [...parseCodes('model.go'), ...parseCodes('codes.go')].filter(([, code]) => code !== 0)

  it('后端错误码没有重号', () => {
    const values = backendCodes.map(([, code]) => code)
    expect(new Set(values).size).toBe(values.length)
  })

  it('每个错误码都有简体中文和英文文案', () => {
    const missing = backendCodes.filter(([, code]) => !(code in errorsZhCN) || !(code in errorsEn))
    expect(missing.map(([name, code]) => `${name}=${code}`)).toEqual([])
  })

  it('双语映射没有多余的错误码', () => {
    const defined = new Set(backendCodes.map(([, code]) => code))
    const extraZh = Object.keys(errorsZhCN).map(Number).filter((code) => !defined.has(code))
    const extraEn = Object.keys(errorsEn).map(Number).filter((code) => !defined.has(code))
    expect({extraZh, extraEn}).toEqual({extraZh: [], extraEn: []})
  })

  it('界面文案的键在两种语言中一致', () => {
    expect(Object.keys(messagesEn).sort()).toEqual(Object.keys(messagesZhCN).sort())
  })

  it('通知默认模板与后端 notifytemplate 一致', () => {
    const source = readFileSync(resolve(RESP_DIR, '../notifytemplate/default.go'), 'utf8')
    const defaults = new Map<string, string>()
    for (const match of source.matchAll(/default(\w+)\s*=\s*"([^"]*)"/g)) {
      defaults.set(match[1], match[2].replace(/\\n/g, '\n'))
    }
    expect(messagesZhCN['template.defaultRemind']).toBe(defaults.get('RemindZhCN'))
    expect(messagesZhCN['template.defaultDue']).toBe(defaults.get('DueZhCN'))
    expect(messagesEn['template.defaultRemind']).toBe(defaults.get('RemindEn'))
    expect(messagesEn['template.defaultDue']).toBe(defaults.get('DueEn'))
  })
})

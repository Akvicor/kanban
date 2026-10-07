import {describe, expect, it} from 'vitest'
import {classifyPaste, setClipboard} from './clipboard'

describe('classifyPaste', () => {
  it('区分没有内容、跨面板、复制和剪切', () => {
    setClipboard(null)
    expect(classifyPaste(null, 1)).toBe('empty')
    expect(classifyPaste({mode: 'copy', cardId: 7, panelId: 2}, 1)).toBe('cross-panel')
    expect(classifyPaste({mode: 'copy', cardId: 7, panelId: 1}, 1)).toBe('copy')
    expect(classifyPaste({mode: 'cut', cardId: 7, panelId: 1}, 1)).toBe('cut')
  })
})

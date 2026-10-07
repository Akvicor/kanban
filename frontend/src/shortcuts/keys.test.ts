import {describe, expect, it} from 'vitest'
import {findOwner, formatKey, keyFromEvent, type KeyInput} from './keys'

function key(code: string, modifiers: Partial<KeyInput> = {}): KeyInput {
  return {code, ctrlKey: false, metaKey: false, altKey: false, shiftKey: false, ...modifiers}
}

describe('keyFromEvent', () => {
  it('按物理按键生成规范的按键字符串', () => {
    expect(keyFromEvent(key('KeyE'))).toBe('E')
    expect(keyFromEvent(key('Digit1', {shiftKey: true}))).toBe('Shift+1')
    expect(keyFromEvent(key('KeyC', {metaKey: true}))).toBe('Mod+C')
    expect(keyFromEvent(key('KeyK', {ctrlKey: true, shiftKey: true, altKey: true}))).toBe('Mod+Alt+Shift+K')
    expect(keyFromEvent(key('NumpadEnter'))).toBe('Enter')
    expect(keyFromEvent(key('F12'))).toBe('F12')
  })

  it('修饰键本身和不能绑定的键返回 null', () => {
    for (const code of ['ShiftLeft', 'ControlLeft', 'MetaRight', 'Escape', 'Tab', 'F13', 'Minus']) {
      expect(keyFromEvent(key(code))).toBeNull()
    }
  })
})

describe('formatKey', () => {
  it('按平台显示修饰键', () => {
    expect(formatKey('Mod+Shift+K', true)).toBe('⌘ + ⇧ + K')
    expect(formatKey('Mod+C', false)).toBe('Ctrl + C')
    expect(formatKey('Space', false)).toBe('空格')
  })
})

describe('findOwner', () => {
  it('找出已绑定按键的操作', () => {
    const bindings = {open_card: ['E', 'Enter'], edit_title: ['T']}
    expect(findOwner(bindings, 'Enter')).toBe('open_card')
    expect(findOwner(bindings, 'Q')).toBeNull()
  })
})

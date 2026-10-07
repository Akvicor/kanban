import {act, renderHook} from '@testing-library/react'
import type {ReactNode} from 'react'
import {MemoryRouter, useNavigate} from 'react-router-dom'
import {afterEach, describe, expect, it, vi} from 'vitest'
import {NARROW_QUERY, useSidebarState} from './sidebar'

function useSidebarWithNavigate() {
  return {state: useSidebarState(), navigate: useNavigate()}
}

const wrapper = ({children}: {children: ReactNode}) => <MemoryRouter>{children}</MemoryRouter>

/** 模拟窗口宽度：narrow 为 true 时处于平板或手机宽度。 */
function setNarrow(narrow: boolean) {
  vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({matches: narrow && query === NARROW_QUERY}) as MediaQueryList)
}

describe('useSidebarState', () => {
  afterEach(() => vi.restoreAllMocks())

  it('电脑宽度下默认收起，可以展开；展开状态不保存在本设备', () => {
    setNarrow(false)
    const first = renderHook(useSidebarState, {wrapper})
    expect(first.result.current.collapsed).toBe(true)
    act(() => first.result.current.control.toggle())
    expect(first.result.current.collapsed).toBe(false)
    expect(first.result.current.open).toBe(false)
    first.unmount()

    // 重新挂载相当于重新打开页面：回到收起，且本机没有保存展开状态。
    const second = renderHook(useSidebarState, {wrapper})
    expect(second.result.current.collapsed).toBe(true)
    expect(localStorage.length).toBe(0)
  })

  it('平板和手机宽度下侧栏滑出后，跳转到其他页面或点遮罩时收回，且不保存', () => {
    setNarrow(true)
    const {result} = renderHook(useSidebarWithNavigate, {wrapper})
    act(() => result.current.state.control.toggle())
    expect(result.current.state.open).toBe(true)

    act(() => result.current.navigate('/files'))
    expect(result.current.state.open).toBe(false)

    act(() => result.current.state.control.toggle())
    act(() => result.current.state.control.close())
    expect(result.current.state.open).toBe(false)
    expect(localStorage.length).toBe(0)
  })
})

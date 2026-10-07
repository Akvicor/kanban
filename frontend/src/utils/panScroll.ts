/**
 * 鼠标拖动平移看板视图。
 *
 * 两种起手方式，都是左键按住后横向拖动，滚动列表行（比用滚动条方便）：
 *   - 空白背景：直接按在列表之外的背景上（列表之间、列表行右侧和下方），不需要修饰键，始终可用。
 *   - 修饰键：按住个人设置中选择的修饰键，在面板中顶部工具栏以外的任意位置按下，卡片、列表头、按钮上同样生效；
 *     设置为空值时关闭这一方式。
 * 手势只响应鼠标，触屏沿用原生滑动。
 */
import {useEffect, type RefObject} from 'react'

/** 设置值（与后端 common/types/panmodifier 一致）对应的事件修饰键属性；空值表示关闭修饰键方式。 */
function modifierKey(value: string): 'ctrlKey' | 'altKey' | 'shiftKey' | null {
  if (value === 'ctrl') return 'ctrlKey'
  if (value === 'alt') return 'altKey'
  if (value === 'shift') return 'shiftKey'
  return null
}

/**
 * 修饰键方式中保持原有交互的区域：顶部的搜索筛选工具栏，以及卡片详情抽屉和它的遮罩。
 * 抽屉用于编辑卡片内容，需要正常的选择文本和点击行为。
 */
const EXCLUDED = '.panel-view > .toolbar, .drawer-frame, .drawer-scrim'

/** 横向移动超过该距离（像素）才算发生了平移，此时吞掉松开后的那次 click，避免顺带打开卡片或触发按钮。 */
const CLICK_SLOP = 3

/** 吞掉手势结束后紧跟的一次 click。click 在抬起后同一轮输入处理中派发，下一个任务时撤销拦截。 */
function swallowNextClick(): void {
  const block = (event: MouseEvent) => {
    event.preventDefault()
    event.stopPropagation()
    window.removeEventListener('click', block, {capture: true})
  }
  window.addEventListener('click', block, {capture: true})
  window.setTimeout(() => window.removeEventListener('click', block, {capture: true}), 0)
}

/**
 * 判断按下点是否落在空白背景上：命中元素是 view 本身，或是 scroller 本身且不在它的横向滚动条上。
 * 滚动条的命中元素同样是 scroller，按在滚动条上时交给原生滚动条拖动。
 */
function onBlank(event: PointerEvent, root: HTMLElement, target: HTMLElement): boolean {
  if (event.target === root) return true
  if (event.target !== target) return false
  const rect = target.getBoundingClientRect()
  return event.clientY - rect.top - target.clientTop < target.clientHeight
}

/**
 * 在 view 区域内启用拖动平移，滚动其内部的 scroller（列表行）。
 * 在捕获阶段接管 pointerdown 并阻止默认行为：浏览器随之不再派发兼容的 mousedown，
 * dnd-kit 的 MouseSensor 不会启动，卡片和列表不会被拖起。
 * 监听绑定在 view 上，scroller 在每次按下时读取，因此 scroller 晚于 view 挂载（加载完成后才渲染）也能生效。
 */
export function usePanScroll(view: RefObject<HTMLElement | null>, scroller: RefObject<HTMLElement | null>, modifier: string): void {
  useEffect(() => {
    const root = view.current
    if (!root) return
    const key = modifierKey(modifier)

    const onDown = (event: PointerEvent) => {
      if (event.pointerType !== 'mouse' || !event.isPrimary || event.button !== 0) return
      const target = scroller.current
      if (!target) return
      const origin = event.target instanceof Element ? event.target : null
      const byModifier = key !== null && event[key] && !origin?.closest(EXCLUDED)
      if (!byModifier && !onBlank(event, root, target)) return
      event.preventDefault()
      event.stopPropagation()

      const startX = event.clientX
      const startScroll = target.scrollLeft
      let moved = false
      root.classList.add('panning')
      const onMove = (move: PointerEvent) => {
        const dx = move.clientX - startX
        if (Math.abs(dx) > CLICK_SLOP) moved = true
        target.scrollLeft = startScroll - dx
      }
      const onUp = () => {
        root.classList.remove('panning')
        window.removeEventListener('pointermove', onMove)
        window.removeEventListener('pointerup', onUp)
        window.removeEventListener('pointercancel', onUp)
        if (moved) swallowNextClick()
      }
      window.addEventListener('pointermove', onMove)
      window.addEventListener('pointerup', onUp)
      window.addEventListener('pointercancel', onUp)
    }
    root.addEventListener('pointerdown', onDown, {capture: true})
    return () => root.removeEventListener('pointerdown', onDown, {capture: true})
  }, [view, scroller, modifier])
}

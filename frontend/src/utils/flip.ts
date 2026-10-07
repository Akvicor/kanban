/**
 * 落点挤开动画（FLIP）。
 *
 * 拖动开始时记下容器内各节点（带 data-flip-key 的列表与卡片）的位置，落点后顺序变化的节点
 * 先瞬移回旧位置再缓动到新位置，视觉上就是两侧的卡片/列表被「挤开」。缓动带回弹（方案 B）。
 * 拖动中的那个节点不参与位移（它由拖拽层控制），改为落点时轻微回弹（.landed）。
 * 尊重系统的「减少动态效果」设置；动画只是装饰，不影响顺序与数据。
 */
import {useLayoutEffect, type RefObject} from 'react'

/** 回弹缓动：末端轻微过冲，像被顶开一下再落定。 */
const OVERSHOOT = 'cubic-bezier(0.34, 1.56, 0.64, 1)'
const DURATION = 220

type Snapshot = Map<string, DOMRect>
type Armed = {snapshot: Snapshot; skipKey: string}

/** 已在拖动中、等待落点播放的容器。 */
const armed = new WeakMap<HTMLElement, Armed>()

function prefersReducedMotion(): boolean {
  return typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
}

/** 记下容器内各节点的当前位置。拖动开始时调用；skipKey 是拖动中的节点，落点回弹交给它。 */
export function armFlip(root: HTMLElement | null, skipKey: string): void {
  if (!root) return
  const snapshot: Snapshot = new Map()
  for (const el of root.querySelectorAll('[data-flip-key]')) {
    snapshot.set(el.getAttribute('data-flip-key') ?? '', el.getBoundingClientRect())
  }
  armed.set(root, {snapshot, skipKey})
}

/** 播放一次挤开动画：位置变化的节点从旧位置缓动到新位置，落点节点加回弹。 */
function playFlip(root: HTMLElement, armedFor: Armed): void {
  const landed = root.querySelector(`[data-flip-key="${armedFor.skipKey}"]`)
  if (landed instanceof HTMLElement) {
    landed.classList.remove('landed')
    void landed.offsetWidth // 重新触发动画
    landed.classList.add('landed')
    landed.addEventListener('animationend', () => landed.classList.remove('landed'), {once: true})
  }
  if (prefersReducedMotion()) return
  for (const el of root.querySelectorAll('[data-flip-key]')) {
    if (!(el instanceof HTMLElement)) continue
    const key = el.getAttribute('data-flip-key') ?? ''
    const before = armedFor.snapshot.get(key)
    if (!before || key === armedFor.skipKey) continue
    const now = el.getBoundingClientRect()
    const dx = before.left - now.left
    const dy = before.top - now.top
    if (Math.abs(dx) < 1 && Math.abs(dy) < 1) continue
    el.style.transition = 'none'
    el.style.transform = `translate(${dx}px, ${dy}px)`
    requestAnimationFrame(() => {
      el.style.transition = `transform ${DURATION}ms ${OVERSHOOT}`
      el.style.transform = ''
      el.addEventListener('transitionend', () => { el.style.transition = '' }, {once: true})
    })
  }
}

/** 解除等待播放的快照：拖动取消或落点不产生顺序变化时调用，避免陈旧快照影响后续动画。 */
export function disarmFlip(root: HTMLElement | null): void {
  if (root) armed.delete(root)
}

/**
 * 在容器的每次布局后检查：有等待播放的快照且确实出现位移时才播放并消耗它。
 * 拖动过程中的重渲染（提示线等）不产生位移，快照保留到落点；动画只是装饰，不影响顺序与数据。
 */
export function useFlip(ref: RefObject<HTMLElement | null>): void {
  useLayoutEffect(() => {
    const root = ref.current
    if (!root) return
    const armedFor = armed.get(root)
    if (!armedFor) return
    if (!hasMovement(root, armedFor)) return
    armed.delete(root)
    try {
      playFlip(root, armedFor)
    } catch {
      // 动画失败忽略，顺序和数据不受影响。
    }
  })
}

/** 容器内是否有节点相对快照发生了位移。 */
function hasMovement(root: HTMLElement, armedFor: Armed): boolean {
  if (prefersReducedMotion()) return false
  for (const el of root.querySelectorAll('[data-flip-key]')) {
    const key = el.getAttribute('data-flip-key') ?? ''
    const before = armedFor.snapshot.get(key)
    if (!before || key === armedFor.skipKey) continue
    const now = el.getBoundingClientRect()
    if (Math.abs(before.left - now.left) >= 1 || Math.abs(before.top - now.top) >= 1) return true
  }
  return false
}

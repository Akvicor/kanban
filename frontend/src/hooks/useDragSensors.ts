import {MouseSensor, TouchSensor, useSensor, useSensors} from '@dnd-kit/core'

/**
 * 拖动的启动方式，各处拖动（列表和卡片、面板标签页、目录、任务）共用：
 * - 鼠标按下后移动 5 像素开始拖动，单纯点击不触发。
 * - 触屏长按 250 毫秒开始拖动，期间允许 8 像素的抖动；长按前手指移动则按滑动处理，用来左右、上下查看。
 *
 * 鼠标和触屏分别用 MouseSensor 和 TouchSensor。dnd-kit 一次只让一个传感器接管，PointerSensor 会先于
 * TouchSensor 接管触摸，使长按失效，因此这里不用 PointerSensor。
 */
export function useDragSensors() {
  return useSensors(
    useSensor(MouseSensor, {activationConstraint: {distance: 5}}),
    useSensor(TouchSensor, {activationConstraint: {delay: 250, tolerance: 8}}),
  )
}

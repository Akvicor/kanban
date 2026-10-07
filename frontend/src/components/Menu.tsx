import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import type {ReactNode} from 'react'
import './Menu.css'

interface MenuProps {
  /** 触发菜单的元素，通常是一个按钮。 */
  trigger: ReactNode
  open?: boolean
  onOpenChange?: (open: boolean) => void
  align?: 'start' | 'end'
  children: ReactNode
}

/** 下拉菜单。键盘导航、焦点管理和点击外部关闭由 Radix DropdownMenu 提供，外观取自当前配色。 */
export function Menu({trigger, open, onOpenChange, align = 'end', children}: MenuProps) {
  return (
    <DropdownMenu.Root open={open} onOpenChange={onOpenChange} modal={false}>
      <DropdownMenu.Trigger asChild>{trigger}</DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        {/* 菜单在 portal 中渲染，但 React 事件沿组件树冒泡，会穿到外面包裹它的链接或按钮上；
            菜单里的点击到此为止，避免点菜单项顺带触发外面的导航或卡片点击。 */}
        <DropdownMenu.Content className="menu" align={align} sideOffset={4} onClick={(event) => event.stopPropagation()}>
          {children}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}

interface MenuItemProps {
  onSelect: () => void
  danger?: boolean
  disabled?: boolean
  children: ReactNode
}

export function MenuItem({onSelect, danger, disabled, children}: MenuItemProps) {
  return (
    <DropdownMenu.Item className={danger ? 'menu-item danger' : 'menu-item'} disabled={disabled} onSelect={onSelect}>
      {children}
    </DropdownMenu.Item>
  )
}

export function MenuSeparator() {
  return <DropdownMenu.Separator className="menu-separator" />
}

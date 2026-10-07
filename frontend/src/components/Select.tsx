import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import {Fragment} from 'react'

export interface SelectOption {
  value: string
  label: string
  /** 分组标题，组名变化时插入一行，对应原生 select 的 optgroup。 */
  group?: string
  disabled?: boolean
}

interface SelectProps {
  value: string
  options: SelectOption[]
  onChange: (value: string) => void
  /** 触发按钮的无障碍名称。 */
  label?: string
  disabled?: boolean
  /** 触发按钮的附加类名，例如 inline-select。 */
  className?: string
}

/**
 * 下拉选择。触发按钮与输入框同框体，展开后是与菜单一致的选项列表，当前项带勾选标记。
 * 原生 select 的箭头和展开面板是浏览器样式，与自绘控件不一致，因此这里自绘。
 */
export function Select({value, options, onChange, label, disabled, className = ''}: SelectProps) {
  const current = options.find((option) => option.value === value)
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button type="button" className={`select-trigger ${className}`.trim()} aria-label={label} disabled={disabled}>
          <span className="select-text">{current?.label ?? ''}</span>
          <svg className="select-chevron" viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d="m6 9 6 6 6-6" />
          </svg>
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        {/* 与菜单同理：选项的点击不穿到外层可能存在的链接或按钮上。 */}
        <DropdownMenu.Content className="menu select-menu" align="start" sideOffset={4} onClick={(event) => event.stopPropagation()}>
          <DropdownMenu.RadioGroup value={value} onValueChange={onChange}>
            {options.map((option, index) => (
              <Fragment key={option.value}>
                {option.group && option.group !== options[index - 1]?.group && (
                  <DropdownMenu.Label className="menu-label">{option.group}</DropdownMenu.Label>
                )}
                <DropdownMenu.RadioItem value={option.value} className="menu-item" disabled={option.disabled}>
                  <span className="menu-check">{option.value === value ? '✓' : ''}</span>
                  {option.label}
                </DropdownMenu.RadioItem>
              </Fragment>
            ))}
          </DropdownMenu.RadioGroup>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}

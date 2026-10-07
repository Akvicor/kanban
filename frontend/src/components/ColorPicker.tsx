import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import {useState, type CSSProperties} from 'react'
import {useT} from '../i18n'

/** 预定义色板：标签、优先级挡位和列表常用的固定颜色，10 个排两行。颜色按色相均匀铺开，
 *  红/粉、橙/黄这两对相邻色另外拉开明度，缩成小色块时也能分辨。 */
export const PRESET_COLORS = [
  '#E5484D', // 红
  '#F76B15', // 橙
  '#F0B100', // 黄
  '#46A758', // 绿
  '#0FA3B1', // 青
  '#3E63DD', // 蓝
  '#8E4EC6', // 紫
  '#F2569B', // 粉
  '#3F4650', // 墨
  '#8B8D98', // 灰
]

interface ColorPickerProps {
  value: string
  onChange: (color: string) => void
  /** 无障碍名称，例如「列表颜色」。 */
  label?: string
}

/**
 * 颜色选择：触发块显示当前颜色，点击弹出面板。面板左侧是一组预定义色板，右侧是取色器，
 * 任选颜色都写入同一个 hex 值。色板与当前值相同时表示选中。
 */
export function ColorPicker({value, onChange, label}: ColorPickerProps) {
  const t = useT()
  const [open, setOpen] = useState(false)
  return (
    <DropdownMenu.Root open={open} onOpenChange={setOpen} modal={false}>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          className="color-trigger"
          style={{'--c': value} as CSSProperties}
          aria-label={label ? t('color.labeled', {label}) : t('common.color')}
          title={value}
        />
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        {/* 面板在 portal 中渲染，点击不穿到外层可能存在的链接或按钮上。 */}
        <DropdownMenu.Content className="menu color-pop" align="start" sideOffset={6} onClick={(event) => event.stopPropagation()}>
          <div className="color-presets">
            {PRESET_COLORS.map((color) => (
              <button
                key={color}
                type="button"
                className="swatch"
                style={{'--c': color} as CSSProperties}
                title={color}
                aria-label={t('color.value', {color})}
                aria-pressed={value.toLowerCase() === color.toLowerCase()}
                onClick={() => {
                  onChange(color)
                  setOpen(false)
                }}
              />
            ))}
          </div>
          <div className="color-custom">
            {/* 取色器：色块显示当前颜色，点击打开系统的取色面板。输入盖在色块上，点击直接落到输入。 */}
            <label className="swatch custom" style={{'--c': value} as CSSProperties} title={t('color.custom')}>
              <input type="color" value={value} aria-label={label ? t('color.pickerLabeled', {label}) : t('color.picker')} onChange={(e) => onChange(e.target.value)} />
            </label>
            <span className="hint">{t('color.picker')}</span>
          </div>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  )
}

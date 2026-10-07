import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import type {CSSProperties} from 'react'
import type {Label, List, PriorityLevel} from '../api/types'
import {EMPTY_FILTER, filterActive, type CardFilter} from '../card/model'
import type {PanelViewState} from './viewState'
import {useT} from '../i18n'

interface PanelToolbarProps {
  view: PanelViewState
  onChange: (changes: Partial<PanelViewState>) => void
  labels: Label[]
  levels: PriorityLevel[]
  /** 面板中未归档的列表。 */
  lists: List[]
}

/** 标签筛选按「不限 → 包含 → 排除 → 不限」循环。 */
function nextLabelMode(mode: 'include' | 'exclude' | undefined): 'include' | 'exclude' | undefined {
  if (mode === undefined) return 'include'
  if (mode === 'include') return 'exclude'
  return undefined
}

function toggle<T>(values: T[], value: T): T[] {
  return values.includes(value) ? values.filter((v) => v !== value) : [...values, value]
}

/**
 * 面板工具栏：搜索、筛选、「只看今日」。列表归档和卡片归档的入口在面板标签页的菜单中。
 * 搜索和筛选条件、开关状态按面板保存在当前设备上。
 */
export function PanelToolbar({view, onChange, labels, levels, lists}: PanelToolbarProps) {
  const t = useT()
  const {filter} = view
  const setFilter = (changes: Partial<CardFilter>) => onChange({filter: {...filter, ...changes}})
  const active = filterActive(filter)

  return (
    <div className="toolbar">
      <label className="search">
        <span aria-hidden>⌕</span>
        <input
          value={filter.query}
          placeholder={t('filter.searchPlaceholder')}
          aria-label={t('filter.search')}
          onChange={(e) => setFilter({query: e.target.value})}
          onKeyDown={(e) => e.key === 'Escape' && setFilter({query: ''})}
        />
      </label>
      <DropdownMenu.Root modal={false}>
        <DropdownMenu.Trigger asChild>
          <button type="button" className={active ? 'chip-btn on' : 'chip-btn'}>
            {t('filter.label')}{active ? t('filter.enabledSuffix') : ''}
          </button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className="menu filter-menu" align="start" sideOffset={4}>
            <DropdownMenu.Label className="menu-label">{t('common.label')}</DropdownMenu.Label>
            {labels.length === 0 && <div className="menu-item hint">{t('card.panelNoLabels')}</div>}
            {labels.map((label) => {
              const mode = filter.labels[label.id]
              return (
                <DropdownMenu.Item
                  key={label.id}
                  className="menu-item"
                  onSelect={(event) => {
                    event.preventDefault()
                    const labelsFilter = {...filter.labels}
                    const next = nextLabelMode(mode)
                    if (next) labelsFilter[label.id] = next
                    else delete labelsFilter[label.id]
                    setFilter({labels: labelsFilter})
                  }}
                >
                  <span className="label" style={{'--c': label.color} as CSSProperties}>
                    {label.name || '\u00a0'}
                  </span>
                  <span className={mode ? `menu-key mode-${mode}` : 'menu-key'}>{mode ? t(mode === 'include' ? 'filter.include' : 'filter.exclude') : t('filter.any')}</span>
                </DropdownMenu.Item>
              )
            })}
            <DropdownMenu.Separator className="menu-separator" />
            <DropdownMenu.Label className="menu-label">{t('common.priority')}</DropdownMenu.Label>
            {[...levels.map((level) => ({id: level.id as number | null, name: level.name})), {id: null, name: t('card.noPriority')}].map((option) => (
              <DropdownMenu.CheckboxItem
                key={option.id ?? 'none'}
                className="menu-item"
                checked={filter.priorities.includes(option.id)}
                onSelect={(event) => event.preventDefault()}
                onCheckedChange={() => setFilter({priorities: toggle(filter.priorities, option.id)})}
              >
                <span className="menu-check">{filter.priorities.includes(option.id) ? '✓' : ''}</span>
                {option.name}
              </DropdownMenu.CheckboxItem>
            ))}
            <DropdownMenu.Separator className="menu-separator" />
            <DropdownMenu.Label className="menu-label">{t('filter.inList')}</DropdownMenu.Label>
            {lists.map((list) => (
              <DropdownMenu.CheckboxItem
                key={list.id}
                className="menu-item"
                checked={filter.lists.includes(list.id)}
                onSelect={(event) => event.preventDefault()}
                onCheckedChange={() => setFilter({lists: toggle(filter.lists, list.id)})}
              >
                <span className="menu-check">{filter.lists.includes(list.id) ? '✓' : ''}</span>
                {list.name}
              </DropdownMenu.CheckboxItem>
            ))}
            {active && (
              <>
                <DropdownMenu.Separator className="menu-separator" />
                <DropdownMenu.Item className="menu-item" onSelect={() => setFilter({labels: EMPTY_FILTER.labels, priorities: [], lists: []})}>
                  {t('filter.clear')}
                </DropdownMenu.Item>
              </>
            )}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
      <label className="switch">
        <input type="checkbox" checked={view.todayOnly} onChange={(e) => onChange({todayOnly: e.target.checked})} />
        <span className="track" />
        {t('filter.todayOnly')}
      </label>
      <span className="toolbar-spacer" />
    </div>
  )
}

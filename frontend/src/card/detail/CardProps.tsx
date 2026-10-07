import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import {useState, type CSSProperties} from 'react'
import * as cardApi from '../../api/card'
import type {Label, NotifyChannel, NotifyDelivery, PriorityLevel} from '../../api/types'
import {Select} from '../../components/Select'
import {formatDateTime, fromZonedInput, toZonedInput} from '../../utils/datetime'
import {EntityType} from '../../sync/store'
import type {NotifyPlace} from '../notifyStatus'
import {NotifyChannelsField} from './NotifyChannelsField'
import {useCardWrite} from './useCardWrite'
import {useT} from '../../i18n'

interface CardPropsProps {
  place: NotifyPlace
  labels: Label[]
  levels: PriorityLevel[]
  channels: NotifyChannel[]
  deliveries: NotifyDelivery[]
  timeZone: string
  readOnly: boolean
  /** 打开卡片时直接展开标签选择（快捷键「编辑卡片标签」）。 */
  openLabels: boolean
}

/**
 * 卡片属性：标签、优先级、提醒和截止时间及通知开关、通知渠道及发送状态，以及只读显示的创建、开始、完成时间。
 * 开始和完成时间由列表的操作配置写入，不能手动修改。
 */
export function CardProps({place, labels, levels, channels, deliveries, timeZone, readOnly, openLabels}: CardPropsProps) {
  const t = useT()
  const {card} = place
  const write = useCardWrite()
  const [labelMenu, setLabelMenu] = useState(openLabels && !readOnly)
  const cardLabels = labels.filter((label) => card.label_ids.includes(label.id))
  const show = (value: string | null) => (value ? formatDateTime(value, timeZone) : '—')

  const saveDates = (changes: Partial<cardApi.CardDatesInput>) =>
    write(EntityType.Card, () =>
      cardApi.setCardDates(card.id, {
        remind_at: card.remind_at,
        due_at: card.due_at,
        remind_notify: card.remind_notify,
        due_notify: card.due_notify,
        ...changes,
      }),
    )

  return (
    <dl className="props">
      <dt>{t('common.label')}</dt>
      <dd>
        {cardLabels.map((label) => (
          <span key={label.id} className="label" style={{'--c': label.color} as CSSProperties}>
            {label.name || '\u00a0'}
          </span>
        ))}
        {!readOnly && (
          <DropdownMenu.Root open={labelMenu} onOpenChange={setLabelMenu} modal={false}>
            <DropdownMenu.Trigger asChild>
              <button type="button" className="add-chip" aria-label={t('card.editLabels')}>
                +
              </button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content className="menu" align="start" sideOffset={4}>
                {labels.length === 0 && <div className="menu-item hint">{t('card.panelNoLabels')}</div>}
                {labels.map((label, index) => (
                  <DropdownMenu.CheckboxItem
                    key={label.id}
                    className="menu-item"
                    checked={card.label_ids.includes(label.id)}
                    onSelect={(event) => event.preventDefault()}
                    onCheckedChange={(on) => void write(EntityType.Card, () => cardApi.setCardLabel(card.id, label.id, on))}
                  >
                    <span className="menu-check">{card.label_ids.includes(label.id) ? '✓' : ''}</span>
                    <span className="label" style={{'--c': label.color} as CSSProperties}>
                      {label.name || '\u00a0'}
                    </span>
                    {index < 10 && <span className="menu-key">{(index + 1) % 10}</span>}
                  </DropdownMenu.CheckboxItem>
                ))}
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        )}
      </dd>

      <dt>{t('common.priority')}</dt>
      <dd>
        <Select
          className="inline-select"
          label={t('common.priority')}
          value={card.priority_level_id === null ? '' : String(card.priority_level_id)}
          disabled={readOnly}
          options={[{value: '', label: t('card.noPriority')}, ...levels.map((level) => ({value: String(level.id), label: level.name}))]}
          onChange={(value) => void write(EntityType.Card, () => cardApi.setCardPriority(card.id, value === '' ? null : Number(value)))}
        />
      </dd>

      <dt>{t('common.remind')}</dt>
      <dd>
        <input
          type="datetime-local"
          className="input inline-input date-input"
          aria-label={t('common.remindAt')}
          disabled={readOnly}
          value={toZonedInput(card.remind_at, timeZone)}
          onChange={(e) => void saveDates({remind_at: fromZonedInput(e.target.value, timeZone)})}
        />
        <label className="notify-toggle">
          <input type="checkbox" checked={card.remind_notify} disabled={readOnly} onChange={(e) => void saveDates({remind_notify: e.target.checked})} />
          {t('common.notify')}
        </label>
      </dd>

      <dt>{t('common.due')}</dt>
      <dd>
        <input
          type="datetime-local"
          className="input inline-input date-input"
          aria-label={t('common.dueAt')}
          disabled={readOnly}
          value={toZonedInput(card.due_at, timeZone)}
          onChange={(e) => void saveDates({due_at: fromZonedInput(e.target.value, timeZone)})}
        />
        <label className="notify-toggle">
          <input type="checkbox" checked={card.due_notify} disabled={readOnly} onChange={(e) => void saveDates({due_notify: e.target.checked})} />
          {t('common.notify')}
        </label>
      </dd>

      <NotifyChannelsField place={place} channels={channels} deliveries={deliveries} timeZone={timeZone} readOnly={readOnly} />

      <dt>{t('common.start')}</dt>
      <dd className="muted">{show(card.started_at)}</dd>
      <dt>{t('common.done')}</dt>
      <dd className="muted">{show(card.completed_at)}</dd>
      <dt>{t('common.create')}</dt>
      <dd className="muted">{show(card.created_at)}</dd>
    </dl>
  )
}

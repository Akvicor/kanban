import * as DropdownMenu from '@radix-ui/react-dropdown-menu'
import {Link} from 'react-router-dom'
import {setCardChannel} from '../../api/notify'
import type {NotifyChannel, NotifyDelivery, NotifyKind} from '../../api/types'
import {useNow} from '../../hooks/useNow'
import {EntityType} from '../../sync/store'
import {formatDateTime} from '../../utils/datetime'
import {deliveryStatus, type DeliveryStatus, type NotifyPlace} from '../notifyStatus'
import {useCardWrite} from './useCardWrite'
import {t, useT} from '../../i18n'

function statusText(status: DeliveryStatus, timeZone: string): string {
  switch (status.state) {
    case 'future':
      return t('notify.notDue')
    case 'waiting':
      return t('notify.waiting')
    case 'sent':
      return t('notify.sentAt', {time: formatDateTime(status.sentAt, timeZone)})
    case 'failed':
      return t('notify.failedRetry', {time: formatDateTime(status.nextAttemptAt, timeZone)})
  }
}

interface NotifyChannelsFieldProps {
  place: NotifyPlace
  channels: NotifyChannel[]
  deliveries: NotifyDelivery[]
  timeZone: string
  readOnly: boolean
}

/**
 * 卡片属性中的「通知渠道」：选择提醒和截止通知发往哪些渠道（下拉多选），
 * 并在每条渠道下显示提醒和截止各自的发送状态。只在当前满足发送条件时显示状态。
 * 渲染为 dl 中的一对 dt、dd。
 */
export function NotifyChannelsField({place, channels, deliveries, timeZone, readOnly}: NotifyChannelsFieldProps) {
  const t = useT()
  const {card} = place
  const write = useCardWrite()
  const now = useNow(30000)
  const selected = channels.filter((channel) => card.notify_channel_ids.includes(channel.id))

  return (
    <>
      <dt>{t('notify.channels')}</dt>
      <dd className="notify-channels">
        {/* 没有选择时只显示加号，它本身就能表示未选择；只读时按其他行的空值显示。 */}
        {selected.length === 0 && readOnly && <span className="muted">—</span>}
        {selected.map((channel) => (
          <div key={channel.id} className="notify-channel">
            <span className="notify-channel-name">{channel.name}</span>
            {(['remind', 'due'] as NotifyKind[]).map((kind) => {
              const status = deliveryStatus(place, kind, channel.id, deliveries, now)
              if (!status) return null
              return (
                <span key={kind} className={`notify-status ${status.state}`} title={status.state === 'failed' ? status.error : undefined}>
                  {kind === 'remind' ? t('common.remind') : t('common.due')}：{statusText(status, timeZone)}
                  {status.state === 'failed' && status.error && <span className="notify-error">{status.error}</span>}
                </span>
              )
            })}
          </div>
        ))}
        {!readOnly && (
          <DropdownMenu.Root modal={false}>
            <DropdownMenu.Trigger asChild>
              <button type="button" className="add-chip" aria-label={t('notify.chooseChannel')}>
                +
              </button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content className="menu" align="start" sideOffset={4}>
                {channels.length === 0 && (
                  <DropdownMenu.Item className="menu-item" asChild>
                    <Link to="/channels">{t('notify.noneCreateOne')}</Link>
                  </DropdownMenu.Item>
                )}
                {channels.map((channel) => {
                  const on = card.notify_channel_ids.includes(channel.id)
                  return (
                    <DropdownMenu.CheckboxItem
                      key={channel.id}
                      className="menu-item"
                      checked={on}
                      onSelect={(event) => event.preventDefault()}
                      onCheckedChange={(checked) => void write(EntityType.Card, () => setCardChannel(card.id, channel.id, checked))}
                    >
                      <span className="menu-check">{on ? '✓' : ''}</span>
                      {channel.name}
                    </DropdownMenu.CheckboxItem>
                  )
                })}
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        )}
      </dd>
    </>
  )
}

import {MenuButton} from '../../layout/MenuButton'
import {useState} from 'react'
import {errorMessage} from '../../api/client'
import {deleteChannel, testChannel} from '../../api/notify'
import type {NotifyChannel} from '../../api/types'
import {ConfirmDialog} from '../../components/ConfirmDialog'
import {useToast} from '../../components/Toast'
import {useNotifyChannels, useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {ChannelDialog} from './ChannelDialog'
import './ChannelsPage.css'
import {useT} from '../../i18n'

type Pending = {type: 'edit'; channel: NotifyChannel | null} | {type: 'delete'; channel: NotifyChannel}

/** 一条渠道测试发送的结果：进行中、成功，或错误信息。 */
type TestResult = {state: 'testing'} | {state: 'ok'} | {state: 'error'; message: string}

/**
 * 通知渠道：用户的全部 gmsg 渠道。卡片在提醒时间和截止时间到达时，通过它选择的渠道发送通知。
 * 可以新建、编辑、删除渠道，以及用渠道发送一条测试消息。Token 和 Sign 不回显。
 */
export function ChannelsPage() {
  const t = useT()
  const store = useSyncStore()
  const channels = useNotifyChannels()
  const {show: showToast, element: toastElement} = useToast()
  const [pending, setPending] = useState<Pending | null>(null)
  const [results, setResults] = useState<Record<number, TestResult>>({})

  async function test(channel: NotifyChannel) {
    setResults((current) => ({...current, [channel.id]: {state: 'testing'}}))
    let result: TestResult = {state: 'ok'}
    try {
      await testChannel(channel.id)
    } catch (err) {
      result = {state: 'error', message: errorMessage(err)}
    }
    setResults((current) => ({...current, [channel.id]: result}))
  }

  async function remove(channel: NotifyChannel) {
    setPending(null)
    try {
      const result = await deleteChannel(channel.id)
      store.applyLocal(EntityType.NotifyChannel, channel.id, null, result.revision)
    } catch (err) {
      showToast(errorMessage(err))
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <MenuButton />
        <h1>{t('notify.channels')}</h1>
        <span className="grow" />
        <button type="button" className="btn primary" onClick={() => setPending({type: 'edit', channel: null})}>
          {t('channels.newChannel')}
        </button>
      </div>
      <p className="hint">{t('channels.hint')}</p>
      <div className="card-section channel-list">
        {channels.length === 0 && <p className="hint">{t('notify.none')}</p>}
        {channels.map((channel) => {
          const result = results[channel.id]
          return (
            <div key={channel.id} className="channel-row">
              <div className="channel-info">
                <strong>{channel.name}</strong>
                <span className="hint">
                  {channel.api} · {channel.format === 'markdown' ? 'Markdown' : t('common.text')} · Token {channel.token_set ? t('common.isSet') : t('common.notSet')} · Sign{' '}
                  {channel.sign_set ? t('common.isSet') : t('common.notSet')}
                </span>
                {result?.state === 'ok' && <span className="form-ok">{t('channels.testSent')}</span>}
                {result?.state === 'error' && <span className="form-error">{result.message}</span>}
              </div>
              <div className="channel-actions">
                <button type="button" className="btn sm" disabled={result?.state === 'testing'} onClick={() => void test(channel)}>
                  {result?.state === 'testing' ? t('channels.sending') : t('channels.testSend')}
                </button>
                <button type="button" className="btn sm" onClick={() => setPending({type: 'edit', channel})}>
                  {t('common.edit')}
                </button>
                <button type="button" className="btn sm danger" onClick={() => setPending({type: 'delete', channel})}>
                  {t('common.delete')}
                </button>
              </div>
            </div>
          )
        })}
      </div>
      {pending?.type === 'edit' && <ChannelDialog channel={pending.channel} onClose={() => setPending(null)} />}
      {pending?.type === 'delete' && (
        <ConfirmDialog
          open
          title={t('channels.deleteTitle', {name: pending.channel.name})}
          description={t('channels.deleteDesc')}
          confirmText={t('common.delete')}
          danger
          onCancel={() => setPending(null)}
          onConfirm={() => void remove(pending.channel)}
        />
      )}
      {toastElement}
    </div>
  )
}

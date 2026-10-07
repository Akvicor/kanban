import {useState, type FormEvent} from 'react'
import * as cardApi from '../../api/card'
import type {Card} from '../../api/types'
import {useNow} from '../../hooks/useNow'
import {EntityType} from '../../sync/store'
import {formatDuration, parseDuration, timerSeconds} from '../model'
import {useCardWrite} from './useCardWrite'
import {useT} from '../../i18n'

/** 定时器：显示当前读数，可以开始、停止，或直接修改累计时间（H:MM:SS）。 */
export function TimerBox({card, readOnly}: {card: Card; readOnly: boolean}) {
  const t = useT()
  const write = useCardWrite()
  const now = useNow(1000)
  const [editing, setEditing] = useState<string | null>(null)
  const [error, setError] = useState('')
  const running = card.timer_started_at !== null

  async function submit(event: FormEvent) {
    event.preventDefault()
    if (editing === null) return
    const seconds = parseDuration(editing)
    if (seconds === null) {
      setError(t('timer.formatHint'))
      return
    }
    if ((await write(EntityType.Card, () => cardApi.cardTimer(card.id, 'set', seconds))) === 'ok') {
      setEditing(null)
      setError('')
    }
  }

  return (
    <section className="section">
      <h5>{t('card.timer')}</h5>
      <div className="timer-box">
        {editing === null ? (
          <>
            <span className="time">{formatDuration(timerSeconds(card, now))}</span>
            <span className="grow" />
            {!readOnly && (
              <>
                <button type="button" className="btn sm ghost" onClick={() => setEditing(formatDuration(timerSeconds(card, Date.now())))}>
                  {t('common.modify')}
                </button>
                <button
                  type="button"
                  className={running ? 'btn sm' : 'btn sm primary'}
                  onClick={() => void write(EntityType.Card, () => cardApi.cardTimer(card.id, running ? 'stop' : 'start'))}
                >
                  {running ? t('common.stop') : t('common.start')}
                </button>
              </>
            )}
          </>
        ) : (
          <form className="timer-edit" onSubmit={submit}>
            <input className="input" value={editing} aria-label={t('timer.total')} autoFocus onChange={(e) => setEditing(e.target.value)} />
            <button type="button" className="btn sm" onClick={() => setEditing(null)}>
              {t('common.cancel')}
            </button>
            <button type="submit" className="btn sm primary">
              {t('common.save')}
            </button>
          </form>
        )}
      </div>
      {error && <p className="form-error">{error}</p>}
    </section>
  )
}

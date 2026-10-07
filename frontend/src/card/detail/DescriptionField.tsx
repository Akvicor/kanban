import {lazy, Suspense, useRef, useState} from 'react'
import * as cardApi from '../../api/card'
import type {Card, EditorMode} from '../../api/types'
import {updateSettings} from '../../api/user'
import {useSettings, useSyncStore} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {useCardWrite} from './useCardWrite'
import {useT} from '../../i18n'

// 渲染和编辑器体积较大，打开卡片和开始编辑时才加载。
const MarkdownView = lazy(() => import('../../markdown/MarkdownView'))
const MarkdownEditor = lazy(() => import('../../markdown/MarkdownEditor'))

/** 描述的最大字符数，与后端一致。 */
const DESCRIPTION_MAX_LENGTH = 1048576

/** 按字符（码点）计算是否超长；UTF-16 长度不超过上限时码点数一定不超过，免去逐字计数。 */
function tooLong(text: string): boolean {
  return text.length > DESCRIPTION_MAX_LENGTH && [...text].length > DESCRIPTION_MAX_LENGTH
}

/**
 * 卡片描述（Markdown）。平时显示渲染结果，点击「编辑」或空描述后打开编辑器。
 * Ctrl/Cmd + Enter、Esc 或「保存」提交，「取消」放弃修改；切换编辑模式后记住，下次直接进入该模式。
 *
 * 描述在编辑期间被其他设备修改过时，服务端返回冲突：编辑器保留草稿，上方提示冲突，
 * 用户可以查看其他设备保存的最新内容，然后选择「仍然保存」覆盖它，或放弃自己的修改。
 * 冲突期间 Ctrl/Cmd + Enter、Esc 和「保存」不提交，避免在不知情时覆盖。
 */
export function DescriptionField({card, readOnly}: {card: Card; readOnly: boolean}) {
  const t = useT()
  const store = useSyncStore()
  const settings = useSettings()
  const write = useCardWrite()
  const [editing, setEditing] = useState(false)
  const [initial, setInitial] = useState('')
  const draft = useRef('')
  const base = useRef(0)
  const [exceeded, setExceeded] = useState(false)
  const [conflict, setConflict] = useState(false)
  const [showLatest, setShowLatest] = useState(false)
  const saving = useRef(false)

  function begin() {
    if (readOnly) return
    base.current = store.versionOf(EntityType.Card, card.id)
    draft.current = card.description
    setInitial(card.description)
    setExceeded(false)
    setEditing(true)
  }

  function close() {
    setEditing(false)
    setConflict(false)
    setShowLatest(false)
  }

  /** 保存草稿。overwrite 为 true 时以本地最新的同步序号提交，覆盖其他设备保存的描述。 */
  async function save(overwrite: boolean) {
    if (saving.current || tooLong(draft.current) || (conflict && !overwrite)) return
    const text = draft.current
    if (text === card.description) {
      close()
      return
    }
    if (overwrite) base.current = store.versionOf(EntityType.Card, card.id)
    saving.current = true
    const outcome = await write(EntityType.Card, () => cardApi.updateCardDescription(card.id, text, base.current))
    saving.current = false
    // 其他失败保留草稿以便重试。
    if (outcome === 'ok') close()
    else if (outcome === 'conflict') setConflict(true)
  }

  function changeMode(mode: EditorMode) {
    if (mode !== settings.editor_mode) {
      void write(EntityType.Settings, () => updateSettings({editor_mode: mode}))
    }
  }

  return (
    <section className="section">
      <h5>
        {t('card.descriptionSection')}
        {!readOnly && !editing && (
          <button type="button" className="btn sm ghost extra" onClick={begin}>
            {t('common.edit')}
          </button>
        )}
      </h5>
      <Suspense fallback={<p className="hint">{t('common.loading')}</p>}>
        {editing ? (
          <>
            {conflict && (
              <div className="conflict-note" role="alert">
                <span>{t('card.descriptionConflict')}</span>
                <button type="button" className="btn sm ghost" onClick={() => setShowLatest(!showLatest)}>
                  {showLatest ? t('card.hideLatest') : t('card.showLatest')}
                </button>
                <button type="button" className="btn sm primary" disabled={exceeded} onClick={() => void save(true)}>
                  {t('card.saveAnyway')}
                </button>
                <button type="button" className="btn sm" onClick={close}>
                  {t('card.discardMine')}
                </button>
              </div>
            )}
            {conflict && showLatest && (
              <div className="conflict-latest" aria-label={t('card.latestDescription')}>
                {card.description ? <MarkdownView source={card.description} /> : <p className="hint">{t('card.latestEmpty')}</p>}
              </div>
            )}
            <MarkdownEditor
              initialValue={initial}
              mode={settings.editor_mode}
              onChange={(value) => {
                draft.current = value
                setExceeded(tooLong(value))
              }}
              onSubmit={() => void save(false)}
              onCancel={() => void save(false)}
              onModeChange={changeMode}
            />
            {!conflict && (
              <div className="form-actions description-actions">
                {exceeded && <p className="form-error">{t('card.descriptionTooLong')}</p>}
                <button type="button" className="btn sm" onClick={close}>
                  {t('common.cancel')}
                </button>
                <button type="button" className="btn sm primary" disabled={exceeded} onClick={() => void save(false)}>
                  {t('common.save')}
                </button>
              </div>
            )}
            {conflict && exceeded && <p className="form-error">{t('card.descriptionTooLong')}</p>}
          </>
        ) : card.description ? (
          <MarkdownView source={card.description} />
        ) : (
          <p className={readOnly ? 'hint' : 'hint description-empty'} onClick={begin}>
            {readOnly ? t('card.noDescription') : t('card.noDescriptionAdd')}
          </p>
        )}
      </Suspense>
    </section>
  )
}

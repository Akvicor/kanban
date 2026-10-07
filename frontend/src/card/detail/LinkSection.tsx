import {useState, type FormEvent} from 'react'
import * as cardApi from '../../api/card'
import type {Board, CardLink, Panel} from '../../api/types'
import {Dialog} from '../../components/Dialog'
import {Select} from '../../components/Select'
import {boardOptions} from '../../directory/tree'
import {boardTabs} from '../../panel/model'
import {useDirectoryData, usePanelData} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {useCardWrite} from './useCardWrite'
import {t, useT} from '../../i18n'

/** 关联目标当前的状态。kind 是判别值，界面显示时映射到 common.board / common.panel。 */
interface LinkTarget {
  kind: 'board' | 'panel'
  name: string
  /** 打开目标的地址；目标已归档或不存在时为 null。 */
  href: string | null
  /** 不能打开时的提示。 */
  notice: string
}

function describeTarget(link: CardLink, boards: Board[], panels: Panel[]): LinkTarget {
  if (link.board_id !== null) {
    const board = boards.find((b) => b.id === link.board_id)
    if (!board) return {kind: 'board', name: t('card.linkBoardGone'), href: null, notice: t('card.linkBoardGone')}
    if (board.archived_at) return {kind: 'board', name: board.name, href: null, notice: t('card.linkBoardArchived')}
    return {kind: 'board', name: board.name, href: `/board/${board.id}`, notice: ''}
  }
  const panel = panels.find((p) => p.id === link.panel_id)
  if (!panel) return {kind: 'panel', name: t('card.linkPanelGone'), href: null, notice: t('card.linkPanelGone')}
  const board = boards.find((b) => b.id === panel.board_id)
  const name = board ? `${board.name} / ${panel.name}` : panel.name
  if (panel.archived_at || !board || board.archived_at) return {kind: 'panel', name, href: null, notice: t('card.linkPanelArchived')}
  return {kind: 'panel', name, href: `/board/${board.id}/panel/${panel.id}`, notice: ''}
}

/**
 * 卡片关联：快速跳转到某个看板或面板的入口，按添加顺序展示。点击在新窗口打开；
 * 目标已归档时不打开，只提示已归档。面板被移到其他看板后，关联跟着面板走。
 */
export function LinkSection({cardId, links, readOnly, onNotice}: {cardId: number; links: CardLink[]; readOnly: boolean; onNotice: (text: string) => void}) {
  const t = useT()
  const write = useCardWrite()
  const {boards} = useDirectoryData()
  const {panels} = usePanelData()
  const [adding, setAdding] = useState(false)
  const own = links.filter((link) => link.card_id === cardId).sort((a, b) => a.position - b.position || a.id - b.id)

  return (
    <section className="section">
      <h5>
        {t('card.linkSection')}
        {!readOnly && (
          <button type="button" className="btn sm ghost extra" onClick={() => setAdding(true)}>
            {t('common.add')}
          </button>
        )}
      </h5>
      {own.length === 0 && <p className="hint">{t('card.noLinks')}</p>}
      {own.map((link) => {
        const target = describeTarget(link, boards, panels)
        return (
          <div
            key={link.id}
            className={target.href ? 'link-row' : 'link-row archived'}
            role="link"
            tabIndex={0}
            onClick={() => (target.href ? window.open(target.href, '_blank', 'noopener') : onNotice(target.notice))}
          >
            <span className="kind">{target.kind === 'board' ? t('common.board') : t('common.panel')}</span>
            <span className="link-name">{target.name}</span>
            {target.notice && <span className="arch">{target.notice}</span>}
            {!readOnly && (
              <button
                type="button"
                className="icon-btn sm"
                aria-label={t('card.removeLink')}
                onClick={(event) => {
                  event.stopPropagation()
                  void write(EntityType.CardLink, () => cardApi.deleteCardLink(link.id), link.id)
                }}
              >
                ×
              </button>
            )}
          </div>
        )
      })}
      {adding && <AddLinkDialog cardId={cardId} onClose={() => setAdding(false)} />}
    </section>
  )
}

/** 选择关联目标：一个看板，或某个看板中的一个面板。 */
function AddLinkDialog({cardId, onClose}: {cardId: number; onClose: () => void}) {
  const write = useCardWrite()
  const {folders, boards} = useDirectoryData()
  const {panels} = usePanelData()
  const options = boardOptions(folders, boards)
  const [value, setValue] = useState(options[0] ? `board:${options[0].board.id}` : '')

  async function submit(event: FormEvent) {
    event.preventDefault()
    const [kind, id] = value.split(':')
    const target = kind === 'board' ? {board_id: Number(id)} : {panel_id: Number(id)}
    if ((await write(EntityType.CardLink, () => cardApi.addCardLink(cardId, target))) === 'ok') onClose()
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={t('card.addLink')} description={t('card.linkHint')}>
      <form className="form" onSubmit={submit}>
        <label className="field">
          <span>{t('common.target')}</span>
          <Select
            label={t('common.target')}
            value={value}
            options={options.flatMap((option) => [
              {value: `board:${option.board.id}`, label: t('link.boardOption', {name: option.board.name}), group: option.label},
              ...boardTabs(panels, option.board.id).map((panel) => ({
                value: `panel:${panel.id}`,
                label: t('link.panelOption', {name: panel.name}),
                group: option.label,
              })),
            ])}
            onChange={setValue}
          />
        </label>
        <div className="form-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.cancel')}
          </button>
          <button type="submit" className="btn primary" disabled={!value}>
            {t('common.add')}
          </button>
        </div>
      </form>
    </Dialog>
  )
}

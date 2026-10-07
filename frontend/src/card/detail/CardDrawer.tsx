import {useEffect, useState} from 'react'
import type {Panel} from '../../api/types'
import {CardDetail, type CardFocus} from './CardDetail'

/** 抽屉滑入、滑出的时长，与 CardDetail.css 中 .drawer-frame 的过渡一致。 */
const SLIDE_MS = 250

interface CardDrawerProps {
  panel: Panel
  /** 地址中打开的卡片，null 表示没有打开。 */
  cardId: number | null
  /** 打开卡片的这次导航的标识。每次打开（包括切换到另一张卡片）都重新挂载卡片详情。 */
  openKey: string
  focus: CardFocus
  onClose: () => void
  onNotice: (text: string) => void
}

/**
 * 卡片详情的抽屉外框：常驻在面板中，打开和关闭时滑入、滑出；在卡片之间切换时保持打开，不重复滑动。
 * 关闭后在滑出期间继续显示刚才的卡片。平板宽度下，抽屉后面有遮罩，点遮罩关闭卡片。
 */
export function CardDrawer({panel, cardId, openKey, focus, onClose, onNotice}: CardDrawerProps) {
  const [shown, setShown] = useState<{cardId: number; key: string} | null>(cardId === null ? null : {cardId, key: openKey})
  // 打开或切换卡片时立即显示新卡片；关闭时保留上一张，滑出后再移除。
  if (cardId !== null && shown?.key !== openKey) {
    setShown({cardId, key: openKey})
  }
  const open = cardId !== null

  useEffect(() => {
    if (open || shown === null) return
    const timer = window.setTimeout(() => setShown(null), SLIDE_MS)
    return () => window.clearTimeout(timer)
  }, [open, shown])

  return (
    <>
      <div className={open ? 'drawer-scrim open' : 'drawer-scrim'} aria-hidden="true" onClick={onClose} />
      <div className={open ? 'drawer-frame open' : 'drawer-frame'}>
        {shown && <CardDetail key={shown.key} panel={panel} cardId={shown.cardId} focus={open ? focus : null} onClose={onClose} onNotice={onNotice} />}
      </div>
    </>
  )
}

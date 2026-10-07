import {createContext, useContext} from 'react'
import type {WriteResult} from '../../api/client'
import {t} from '../../i18n'

/**
 * 一次写入的结果：成功、编辑冲突，或其他失败。
 * 编辑冲突时本地数据已更新为服务端的最新内容，不显示提示，由调用方在原位提示并保留草稿，
 * 用户可以覆盖（以最新的同步序号重新提交）或放弃。
 */
export type WriteOutcome = 'ok' | 'conflict' | 'error'

/**
 * 卡片详情中的写入：执行请求，用响应立即更新本地数据；编辑冲突以外的失败显示提示。
 * type 是响应实体的同步类型；响应数据为数组时（批量新建任务）逐项应用。
 * deletedId 给出时表示这是删除，成功后从本地移除该实体（随之删除的下级通过推送到达）。
 */
export type CardWrite = (type: string, request: () => Promise<WriteResult<unknown>>, deletedId?: number) => Promise<WriteOutcome>

export const CardWriteContext = createContext<CardWrite | null>(null)

export function useCardWrite(): CardWrite {
  const write = useContext(CardWriteContext)
  if (!write) {
    throw new Error(t('error.useCardWrite'))
  }
  return write
}

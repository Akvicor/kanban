import {useState} from 'react'
import * as panelApi from '../api/panel'
import {errorMessage, type WriteResult} from '../api/client'
import type {Panel} from '../api/types'
import {Dialog} from '../components/Dialog'
import {usePanelData, useSyncStore} from '../sync/hooks'
import {EntityType} from '../sync/store'
import {panelLabels, panelPriorityLevels} from './model'
import {OptionList} from './OptionList'
import './PanelOptionsDialog.css'
import {useT} from '../i18n'

/** 名称长度上限，与后端一致。 */
const LABEL_NAME_MAX = 50
const PRIORITY_NAME_MAX = 20

interface PanelOptionsDialogProps {
  panel: Panel
  onClose: () => void
}

/**
 * 面板的标签和优先级挡位。二者只属于这个面板，不与其他面板共用。
 * 挡位从上到下优先级由高到低；删除挡位后，使用它的卡片改为没有优先级。
 */
export function PanelOptionsDialog({panel, onClose}: PanelOptionsDialogProps) {
  const t = useT()
  const store = useSyncStore()
  const {labels, priorityLevels} = usePanelData()
  const [error, setError] = useState('')

  /** 执行一次写入，用响应立即更新本地数据；删除时 data 为 null。 */
  const run =
    (type: string) =>
    async (request: () => Promise<WriteResult<unknown>>, deletedId?: number): Promise<void> => {
      setError('')
      try {
        const result = await request()
        const data = result.data as {id: number} | undefined
        store.applyLocal(type, deletedId ?? data!.id, deletedId === undefined ? data : null, result.revision)
      } catch (err) {
        setError(errorMessage(err))
        throw err
      }
    }
  const label = run(EntityType.Label)
  const priority = run(EntityType.PriorityLevel)
  const quiet = (promise: Promise<void>) => promise.catch(() => {})

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()} title={t('panel.optionsTitle', {name: panel.name})}>
      <div className="panel-options">
        <section>
          <h3>{t('common.label')}</h3>
          <p className="hint">{t('panel.labelsHint')}</p>
          <OptionList
            items={panelLabels(labels, panel.id)}
            nameRequired={false}
            nameMaxLength={LABEL_NAME_MAX}
            defaultColor="#3E63DD"
            placeholder={t('panel.labelNamePlaceholder')}
            onCreate={(name, color) => quiet(label(() => panelApi.createLabel(panel.id, name, color)))}
            onUpdate={(id, name, color) => label(() => panelApi.updateLabel(id, name, color))}
            onMove={(id, index) => quiet(label(() => panelApi.reorderLabel(id, index)))}
            onDelete={(id) => quiet(label(() => panelApi.deleteLabel(id), id))}
          />
        </section>
        <section>
          <h3>{t('common.priority')}</h3>
          <p className="hint">{t('panel.levelsHint')}</p>
          <OptionList
            items={panelPriorityLevels(priorityLevels, panel.id)}
            nameRequired
            nameMaxLength={PRIORITY_NAME_MAX}
            defaultColor="#8B8D98"
            placeholder={t('panel.levelName')}
            onCreate={(name, color) => quiet(priority(() => panelApi.createPriority(panel.id, name, color)))}
            onUpdate={(id, name, color) => priority(() => panelApi.updatePriority(id, name, color))}
            onMove={(id, index) => quiet(priority(() => panelApi.reorderPriority(id, index)))}
            onDelete={(id) => quiet(priority(() => panelApi.deletePriority(id), id))}
          />
        </section>
        {error && <p className="form-error">{error}</p>}
        <div className="form-actions">
          <button type="button" className="btn" onClick={onClose}>
            {t('common.done')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}

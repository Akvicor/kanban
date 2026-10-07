import {Dialog} from './Dialog'
import {useT} from '../i18n'

interface ConfirmDialogProps {
  open: boolean
  title: string
  description: string
  confirmText: string
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

/** 需要二次确认的操作。 */
export function ConfirmDialog({open, title, description, confirmText, danger, onConfirm, onCancel}: ConfirmDialogProps) {
  const t = useT()
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onCancel()} title={title} description={description}>
      <div className="form-actions">
        <button type="button" className="btn" onClick={onCancel}>
          {t('common.cancel')}
        </button>
        <button type="button" className={danger ? 'btn danger solid' : 'btn primary'} onClick={onConfirm}>
          {confirmText}
        </button>
      </div>
    </Dialog>
  )
}

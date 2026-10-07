import * as RadixDialog from '@radix-ui/react-dialog'
import type {ReactNode} from 'react'
import './Dialog.css'

interface DialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  /** 附加在弹窗上的类名，用于调整尺寸，例如附件预览的大号弹窗。 */
  className?: string
  children: ReactNode
}

/** 居中弹窗。焦点管理、Esc 关闭和点击遮罩关闭由 Radix Dialog 提供，外观取自当前配色。 */
export function Dialog({open, onOpenChange, title, description, className, children}: DialogProps) {
  return (
    <RadixDialog.Root open={open} onOpenChange={onOpenChange}>
      <RadixDialog.Portal>
        <RadixDialog.Overlay className="dialog-overlay" />
        <RadixDialog.Content className={className ? `dialog ${className}` : 'dialog'}>
          <RadixDialog.Title className="dialog-title">{title}</RadixDialog.Title>
          {description ? (
            <RadixDialog.Description className="dialog-description">{description}</RadixDialog.Description>
          ) : (
            <RadixDialog.Description className="visually-hidden">{title}</RadixDialog.Description>
          )}
          {children}
        </RadixDialog.Content>
      </RadixDialog.Portal>
    </RadixDialog.Root>
  )
}

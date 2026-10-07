import {t} from '../i18n'
/** 快捷键操作在设置界面中的名称。操作 ID 和展示顺序来自后端 /api/user/me。 */

/** 操作 ID 到界面名称的映射，按当前语言返回。 */
function labels(): Record<string, string> {
  return {
    open_card: t('shortcut.openCard'),
    edit_labels: t('shortcut.editLabels'),
    edit_title: t('shortcut.editTitle'),
    archive_card: t('shortcut.archiveCard'),
    copy_card: t('shortcut.copyCard'),
    cut_card: t('shortcut.cutCard'),
    paste_card: t('shortcut.pasteCard'),
  }
}

/** 每个操作最多绑定的按键数，与后端一致。 */
export const MAX_KEYS_PER_ACTION = 3

export function actionLabel(action: string): string {
  const toggle = /^toggle_label_(\d+)$/.exec(action)
  if (toggle) {
    return t('shortcut.toggleLabelN', {n: toggle[1]})
  }
  return labels()[action] ?? action
}

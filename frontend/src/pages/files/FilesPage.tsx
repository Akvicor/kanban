import {MenuButton} from '../../layout/MenuButton'
import {Select} from '../../components/Select'
import {useState} from 'react'
import {errorMessage} from '../../api/client'
import {deleteUserFile, userFileThumbnailUrl, userFileUrl} from '../../api/file'
import type {UserFile} from '../../api/types'
import {download} from '../../attachment/AttachmentItem'
import {extensionOf, previewKind} from '../../attachment/kind'
import {openGallery} from '../../attachment/preview/gallery'
import {PreviewDialog, type PreviewTarget} from '../../attachment/preview/PreviewDialog'
import {ConfirmDialog} from '../../components/ConfirmDialog'
import {useToast} from '../../components/Toast'
import {useSettings, useSyncStore, useUserFiles} from '../../sync/hooks'
import {EntityType} from '../../sync/store'
import {formatDateTime} from '../../utils/datetime'
import {formatSize} from '../../utils/size'
import {sortFiles, type FileSort} from './sortFiles'
import './FilesPage.css'
import {useT} from '../../i18n'

/**
 * 文件管理：用户的全部文件。附件引用的文件在这里统一管理，删除附件只去掉引用，文件本身保留在这里。
 * 只有不再被任何附件引用的文件可以删除；删除后，系统中没有其他用户引用的文件内容随之清除。
 */
export function FilesPage() {
  const t = useT()
  const store = useSyncStore()
  const settings = useSettings()
  const {files, loaded, error} = useUserFiles()
  const {show: showToast, element: toastElement} = useToast()
  const [sort, setSort] = useState<FileSort>('unused')
  const [deleting, setDeleting] = useState<UserFile | null>(null)
  const [preview, setPreview] = useState<PreviewTarget | null>(null)
  const sorted = sortFiles(files, sort)
  const unused = files.filter((file) => file.reference_count === 0)
  const time = (value: string | null) => (value ? formatDateTime(value, settings.timezone) : '—')

  function open(file: UserFile) {
    const kind = previewKind(file, file.name)
    if (kind === 'image') {
      const images = sorted.filter((f) => f.image)
      openGallery(
        images.map((image) => ({src: userFileUrl(image.id), width: image.width, height: image.height, name: image.name})),
        images.indexOf(file),
      )
    } else {
      setPreview({kind, url: userFileUrl(file.id), name: file.name, size: file.size})
    }
  }

  async function remove(file: UserFile) {
    setDeleting(null)
    try {
      const result = await deleteUserFile(file.id)
      store.applyLocal(EntityType.UserFile, file.id, null, result.revision)
    } catch (err) {
      showToast(errorMessage(err))
    }
  }

  return (
    <div className="page">
      <div className="page-header">
        <MenuButton />
        <h1>{t('nav.files')}</h1>
        <label className="field-inline">
          <span>{t('files.sort')}</span>
          <Select
            className="inline-select"
            label={t('files.sort')}
            value={sort}
            options={[
              {value: 'unused', label: t('files.sortUnreferenced')},
              {value: 'first', label: t('files.sortFirstRef')},
            ]}
            onChange={(value) => setSort(value as FileSort)}
          />
        </label>
      </div>
      <p className="hint">
        {t('files.hint', {total: files.length, unused: unused.length, size: formatSize(unused.reduce((total, file) => total + file.size, 0))})}
      </p>
      {error && <p className="form-error">{error}</p>}
      {!loaded && !error && <p className="hint">{t('common.loading')}</p>}
      {loaded && files.length === 0 && <p className="hint">{t('files.none')}</p>}
      {loaded && files.length > 0 && (
        <div className="card-section file-table">
          <div className="file-row file-head">
            <span />
            <span>{t('common.name')}</span>
            <span>{t('files.size')}</span>
            <span>{t('files.refCount')}</span>
            <span>{t('files.firstRef')}</span>
            <span>{t('files.refDropped')}</span>
            <span />
          </div>
          {sorted.map((file) => (
            <div key={file.id} className={file.reference_count === 0 ? 'file-row unused' : 'file-row'}>
              <button type="button" className="file-thumb" aria-label={t('common.openName', {name: file.name})} onClick={() => open(file)}>
                {file.image ? (
                  <img src={userFileThumbnailUrl(file.id, 360)} alt="" loading="lazy" />
                ) : (
                  <span>{extensionOf(file.name).slice(0, 4).toUpperCase() || '—'}</span>
                )}
              </button>
              <button type="button" className="file-name" onClick={() => open(file)}>
                {file.name}
              </button>
              <span className="file-cell" data-label={t('files.size')}>{formatSize(file.size)}</span>
              <span className="file-cell" data-label={t('files.refCount')}>{file.reference_count}</span>
              <span className="file-cell" data-label={t('files.firstRef')}>{time(file.first_referenced_at)}</span>
              <span className="file-cell" data-label={t('files.refDropped')}>{time(file.zero_at)}</span>
              <span className="file-actions">
                <button type="button" className="btn sm" onClick={() => download(userFileUrl(file.id), file.name)}>
                  {t('common.download')}
                </button>
                <button
                  type="button"
                  className="btn sm danger"
                  disabled={file.reference_count > 0}
                  title={file.reference_count > 0 ? t('files.stillReferenced') : undefined}
                  onClick={() => setDeleting(file)}
                >
                  {t('common.delete')}
                </button>
              </span>
            </div>
          ))}
        </div>
      )}
      {deleting && (
        <ConfirmDialog
          open
          title={t('files.deleteTitle', {name: deleting.name})}
          description={t('files.deleteDesc')}
          confirmText={t('common.delete')}
          danger
          onCancel={() => setDeleting(null)}
          onConfirm={() => void remove(deleting)}
        />
      )}
      {preview && <PreviewDialog target={preview} onClose={() => setPreview(null)} />}
      {toastElement}
    </div>
  )
}

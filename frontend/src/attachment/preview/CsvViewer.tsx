import Papa from 'papaparse'
import {useEffect, useMemo, useState} from 'react'
import {CSV_PREVIEW_LIMIT} from '../kind'
import {fetchText, type TextResult} from './fetchText'
import {useT} from '../../i18n'

/** 每页显示的行数，与参考项目一致。 */
const PAGE_SIZE = 50

/** CSV 预览：第一行作表头，每页 50 行。文件必须是 UTF-8，大小不超过 10MB。 */
export default function CsvViewer({url, size}: {url: string; size: number}) {
  const t = useT()
  const [result, setResult] = useState<TextResult | null>(null)
  const [page, setPage] = useState(0)

  useEffect(() => {
    let cancelled = false
    void fetchText(url, size, CSV_PREVIEW_LIMIT).then((value) => !cancelled && setResult(value))
    return () => {
      cancelled = true
    }
  }, [url, size])

  const rows = useMemo(() => (result?.ok ? Papa.parse<string[]>(result.text.trim(), {skipEmptyLines: true}).data : []), [result])

  if (!result) return <p className="hint">{t('common.loading')}</p>
  if (!result.ok) return <p className="hint">{result.reason}</p>
  if (rows.length === 0) return <p className="hint">{t('attachment.emptyFile')}</p>

  const [header, ...body] = rows
  const pages = Math.max(1, Math.ceil(body.length / PAGE_SIZE))
  const current = Math.min(page, pages - 1)
  const visible = body.slice(current * PAGE_SIZE, (current + 1) * PAGE_SIZE)
  return (
    <div className="csv-preview">
      <div className="csv-scroll">
        <table>
          <thead>
            <tr>
              {header.map((cell, i) => (
                <th key={i}>{cell}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {visible.map((row, r) => (
              <tr key={r}>
                {header.map((_, i) => (
                  <td key={i}>{row[i] ?? ''}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {pages > 1 && (
        <div className="csv-pager">
          <button type="button" className="btn sm" disabled={current === 0} onClick={() => setPage(current - 1)}>
            {t('preview.pagePrev')}
          </button>
          <span className="hint">
            {t('files.pageInfo', {page: current + 1, pages, rows: body.length})}
          </span>
          <button type="button" className="btn sm" disabled={current >= pages - 1} onClick={() => setPage(current + 1)}>
            {t('preview.pageNext')}
          </button>
        </div>
      )}
    </div>
  )
}

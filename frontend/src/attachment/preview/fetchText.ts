import {t} from '../../i18n'
/** 读取文本内容的结果：成功时为文字，否则为不能显示的原因。 */
export type TextResult = {ok: true; text: string} | {ok: false; reason: string}

/**
 * 读取文件内容并按 UTF-8 解码。超过 limit 字节或不是有效的 UTF-8 时不显示。
 * 文件接口靠文件 Cookie 认证，因此带上同源凭据。
 */
export async function fetchText(url: string, size: number, limit: number): Promise<TextResult> {
  if (size > limit) {
    return {ok: false, reason: t('preview.fileTooLarge', {size: Math.round(limit / 1024)})}
  }
  let response: Response
  try {
    response = await fetch(url, {credentials: 'same-origin'})
  } catch {
    return {ok: false, reason: t('attachment.networkError')}
  }
  if (!response.ok) {
    return {ok: false, reason: t('preview.readFailed', {status: response.status})}
  }
  try {
    return {ok: true, text: new TextDecoder('utf-8', {fatal: true}).decode(await response.arrayBuffer())}
  } catch {
    return {ok: false, reason: t('attachment.notUtf8')}
  }
}

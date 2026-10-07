/**
 * 后端接口的请求封装。
 *
 * 接口统一返回 {code, msg, data, revision}，HTTP 状态码为 200，结果由 code 区分，取值与后端 common/resp 一致。
 * 业务错误只携带细粒度错误码，展示文案按码从语言包取（src/i18n）；msg 不承载文案。
 * revision 是这次写入产生的同步序号，没有产生变更时省略，见 src/sync/store.ts。
 * 登录令牌保存在本设备，通过 Authorization: Bearer 发送。
 */
import {errorText, t} from '../i18n'

/** 接口结果码（协议级）。细粒度业务错误码见后端 common/resp。 */
export const ResultCode = {
  Succeeded: 0,
  Failed: 1,
  NotFound: 2,
  BadRequest: 3,
  Unauthorized: 4,
  Forbidden: 5,
  Conflict: 6,
  TooManyRequests: 7,
} as const

/** 会话失效类错误码：收到后清除令牌并回到登录页。 */
const SESSION_CODES = new Set([ResultCode.Unauthorized, 1002, 1003])

/**
 * 接口返回非成功结果时抛出的错误。展示文案用 errorMessage 按错误码取；
 * message 只在网络层自身出错时填，直接展示。
 * 编辑冲突时 data 是对象的当前数据，revision 是读取它时的同步序号；其他错误没有这两项。
 */
export class ApiError extends Error {
  readonly code: number
  readonly data: unknown
  readonly revision: number

  constructor(code: number, message = '', data?: unknown, revision = 0) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.data = data
    this.revision = revision
  }
}

interface Envelope<T> {
  code: number
  msg?: string
  data?: T
  revision?: number
}

/** 写入接口的结果：响应数据，以及这次写入产生的同步序号（没有变更时为 0）。 */
export interface WriteResult<T> {
  data: T
  revision: number
}

/** 本设备保存登录令牌的键。 */
export const TOKEN_STORAGE_KEY = 'kanban.token'

export function getToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_STORAGE_KEY)
  } catch {
    return null
  }
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_STORAGE_KEY, token)
}

export function clearToken(): void {
  try {
    localStorage.removeItem(TOKEN_STORAGE_KEY)
  } catch {
    // 存储不可用时令牌本来也没有保存下来。
  }
}

let unauthorizedHandler: (() => void) | null = null

/** 注册登录失效时的处理函数。任何请求返回未登录时，先清除令牌再调用它。 */
export function onUnauthorized(handler: (() => void) | null): void {
  unauthorizedHandler = handler
}

async function request<T>(method: 'GET' | 'POST', path: string, body?: unknown): Promise<WriteResult<T>> {
  const headers: Record<string, string> = {}
  const token = getToken()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }
  // Blob 作为原始字节发送（上传分片），其他请求体按 JSON 发送。
  const raw = body instanceof Blob
  if (body !== undefined) {
    headers['Content-Type'] = raw ? 'application/octet-stream' : 'application/json'
  }

  let response: Response
  try {
    response = await fetch(`/api${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : raw ? body : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(ResultCode.Failed, t('error.network'))
  }

  let envelope: Envelope<T>
  try {
    envelope = (await response.json()) as Envelope<T>
  } catch {
    throw new ApiError(ResultCode.Failed, t('error.badResponse', {status: response.status}))
  }
  if (envelope.code === ResultCode.Succeeded) {
    return {data: envelope.data as T, revision: envelope.revision ?? 0}
  }
  if (SESSION_CODES.has(envelope.code)) {
    clearToken()
    unauthorizedHandler?.()
  }
  throw new ApiError(envelope.code, '', envelope.data, envelope.revision ?? 0)
}

export const http = {
  get: async <T>(path: string) => (await request<T>('GET', path)).data,
  post: async <T = void>(path: string, body: unknown = {}) => (await request<T>('POST', path, body)).data,
  /** 修改同步数据的写入，返回数据和同步序号，调用方用 SyncStore.applyLocal 立即更新本地数据。 */
  write: <T = void>(path: string, body: unknown = {}) => request<T>('POST', path, body),
  /** 以原始字节为请求体的 POST，用于上传分片。 */
  raw: async <T>(path: string, body: Blob) => (await request<T>('POST', path, body)).data,
}

/** 把任意错误转换为可以展示给用户的文字。 */
export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message || errorText(error.code)
  }
  return t('error.unknown')
}

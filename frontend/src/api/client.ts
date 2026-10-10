/**
 * 后端接口的请求封装。
 *
 * 接口统一返回 {code, msg, data, revision}，HTTP 状态码为 200，结果由 code 区分，取值与后端 common/resp 一致。
 * 业务错误只携带细粒度错误码，展示文案按码从语言包取（src/i18n）；msg 不承载文案。
 * revision 是这次写入产生的同步序号，没有产生变更时省略，见 src/sync/store.ts。
 * 登录令牌取当前账号的令牌（见 session/accounts/storage.ts），通过 Authorization: Bearer 发送；
 * 对指定账号的操作（例如退出其他账号）可以显式传入该账号的令牌。
 */
import {errorText, t} from '../i18n'
import {currentToken, expireToken} from '../session/accounts/storage'

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

/** 会话失效类错误码：未登录、登录已失效、请先登录、账号已停用。收到后该令牌的账号标记为已失效。 */
const SESSION_CODES = new Set<number>([ResultCode.Unauthorized, 1002, 1003, 1004])

/** 错误是否表示登录已失效。 */
export function isSessionError(error: unknown): boolean {
  return error instanceof ApiError && SESSION_CODES.has(error.code)
}

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

/** 请求选项。token 显式指定使用的令牌，省略时使用当前账号的令牌。 */
export interface RequestOptions {
  token?: string | null
}

let unauthorizedHandler: (() => void) | null = null

/** 注册当前账号登录失效时的处理函数。请求返回会话失效码时，先把该账号标记为已失效再调用它。 */
export function onUnauthorized(handler: (() => void) | null): void {
  unauthorizedHandler = handler
}

let pending = 0

/** 尚未结束的接口请求数。切换或退出账号会刷新页面，有未结束的请求时不允许进行。 */
export function pendingRequestCount(): number {
  return pending
}

async function request<T>(method: 'GET' | 'POST', path: string, body?: unknown, options: RequestOptions = {}): Promise<WriteResult<T>> {
  pending += 1
  try {
    return await send<T>(method, path, body, options.token === undefined ? currentToken() : options.token)
  } finally {
    pending -= 1
  }
}

async function send<T>(method: 'GET' | 'POST', path: string, body: unknown, token: string | null): Promise<WriteResult<T>> {
  const headers: Record<string, string> = {}
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
  if (SESSION_CODES.has(envelope.code) && token) {
    // 只有失效的是当前账号的令牌时才通知；显式使用其他账号令牌的请求只标记该账号。
    const wasCurrent = token === currentToken()
    expireToken(token)
    if (wasCurrent) unauthorizedHandler?.()
  }
  throw new ApiError(envelope.code, '', envelope.data, envelope.revision ?? 0)
}

export const http = {
  get: async <T>(path: string) => (await request<T>('GET', path)).data,
  post: async <T = void>(path: string, body: unknown = {}, options?: RequestOptions) => (await request<T>('POST', path, body, options)).data,
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

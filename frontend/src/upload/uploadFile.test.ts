import {afterEach, describe, expect, it, vi} from 'vitest'
import {ApiError, ResultCode} from '../api/client'
import {CHUNK_SIZE, uploadFile} from './uploadFile'

/** 按顺序返回接口响应，并记录请求地址。 */
function mockFetch(...responses: object[]) {
  const fetchMock = vi.fn(async () => new Response(JSON.stringify(responses.shift() ?? {code: ResultCode.Succeeded})))
  vi.stubGlobal('fetch', fetchMock)
  return {
    paths: () => fetchMock.mock.calls.map((call) => (call as unknown as [string])[0]),
    body: (index: number) => (fetchMock.mock.calls[index] as unknown as [string, RequestInit])[1].body,
  }
}

const ok = (data: object) => ({code: ResultCode.Succeeded, data})
const SHA_OF_HELLO = '185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969'

describe('uploadFile', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('先在本地计算 sha256，系统已有时不上传内容', async () => {
    const api = mockFetch(ok({exists: true, session_id: 0, received: 0, done: false}))
    const sha = await uploadFile(new File(['Hello'], 'a.txt'), () => {})
    expect(sha).toBe(SHA_OF_HELLO)
    expect(api.paths()).toEqual(['/api/upload/prepare'])
    expect(JSON.parse(api.body(0) as string)).toEqual({sha256: SHA_OF_HELLO, size: 5})
  })

  it('从服务端已收到的位置续传', async () => {
    const api = mockFetch(ok({exists: false, session_id: 7, received: 3, done: false}), ok({exists: false, session_id: 7, received: 5, done: true}))
    await uploadFile(new File(['Hello'], 'a.txt'), () => {})
    expect(api.paths()).toEqual(['/api/upload/prepare', '/api/upload/chunk?session_id=7&offset=3'])
    expect(await (api.body(1) as Blob).text()).toBe('lo')
  })

  it('分片偏移冲突时重新查询已收到的位置后继续', async () => {
    const size = CHUNK_SIZE + 10
    const api = mockFetch(
      ok({exists: false, session_id: 9, received: 0, done: false}),
      {code: ResultCode.Conflict, msg: '分片偏移不一致'},
      ok({exists: false, session_id: 9, received: CHUNK_SIZE, done: false}),
      ok({exists: false, session_id: 9, received: size, done: true}),
    )
    vi.useFakeTimers({toFake: ['setTimeout']})
    const done = uploadFile(new File([new Uint8Array(size)], 'big.bin'), () => {})
    await vi.runAllTimersAsync()
    await done
    vi.useRealTimers()
    expect(api.paths()).toEqual([
      '/api/upload/prepare',
      '/api/upload/chunk?session_id=9&offset=0',
      '/api/upload/prepare',
      `/api/upload/chunk?session_id=9&offset=${CHUNK_SIZE}`,
    ])
  })

  it('校验失败等其他错误直接失败', async () => {
    mockFetch(ok({exists: false, session_id: 1, received: 0, done: false}), {code: ResultCode.BadRequest})
    await expect(uploadFile(new File(['x'], 'x.txt'), () => {})).rejects.toThrow(ApiError)
  })
})

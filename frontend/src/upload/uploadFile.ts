import {ApiError, ResultCode} from '../api/client'
import {prepareUpload, uploadChunk} from '../api/file'
import {hashFile} from './sha256'
import {t} from '../i18n'

/** 单个文件的大小上限，与服务端一致。 */
export const MAX_FILE_SIZE = 4 * 1024 ** 3
/** 每个分片的大小。服务端单片上限为 16MiB。 */
export const CHUNK_SIZE = 8 << 20
/** 网络错误等可重试的失败，每个分片最多重试的次数。 */
const MAX_RETRIES = 3

/** 上传的阶段：计算 sha256、上传内容。 */
export type UploadPhase = 'hashing' | 'uploading'

export interface UploadProgress {
  phase: UploadPhase
  /** 当前阶段已完成的比例，0–1。 */
  ratio: number
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

/** 偏移不一致或网络失败时可以重试；其他错误（例如校验失败、文件过大）直接失败。 */
function retryable(error: unknown): boolean {
  return error instanceof ApiError && (error.code === ResultCode.Conflict || error.code === ResultCode.Failed)
}

/**
 * 按上传协议上传文件，返回 sha256：
 * 先在本地计算 sha256，系统已有这份文件时不上传内容；否则从服务端已收到的位置开始按片上传，
 * 中断后再次上传同一文件时自动续传。
 */
export async function uploadFile(file: File, onProgress: (progress: UploadProgress) => void): Promise<string> {
  if (file.size > MAX_FILE_SIZE) {
    throw new ApiError(ResultCode.BadRequest, t('upload.tooLarge'))
  }
  const sha = await hashFile(file, (ratio) => onProgress({phase: 'hashing', ratio}))
  let state = await prepareUpload(sha, file.size)
  if (state.exists) {
    onProgress({phase: 'uploading', ratio: 1})
    return sha
  }
  let offset = state.received
  let failures = 0
  // 空文件也要发送一次（空的）分片，服务端据此完成上传。
  while (!state.done) {
    onProgress({phase: 'uploading', ratio: file.size === 0 ? 0 : offset / file.size})
    try {
      state = await uploadChunk(state.session_id, offset, file.slice(offset, offset + CHUNK_SIZE))
      offset = state.received
      failures = 0
    } catch (error) {
      if (!retryable(error) || ++failures > MAX_RETRIES) throw error
      await sleep(500 * failures)
      // 重新查询服务端已收到的位置后继续；期间别的上传已完成同一文件时直接结束。
      state = await prepareUpload(sha, file.size)
      if (state.exists) break
      offset = state.received
    }
  }
  onProgress({phase: 'uploading', ratio: 1})
  return sha
}

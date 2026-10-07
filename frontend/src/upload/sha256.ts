import {createSHA256} from 'hash-wasm'

/** 计算 sha256 时每次读取的字节数。分块读取，4GiB 的文件也不需要整体读入内存。 */
const READ_SIZE = 4 << 20

/** 流式计算文件的 sha256（小写十六进制）。onProgress 收到已处理的比例（0–1）。 */
export async function hashFile(file: Blob, onProgress?: (ratio: number) => void): Promise<string> {
  const hasher = await createSHA256()
  hasher.init()
  for (let offset = 0; offset < file.size; offset += READ_SIZE) {
    const chunk = await file.slice(offset, offset + READ_SIZE).arrayBuffer()
    hasher.update(new Uint8Array(chunk))
    onProgress?.(Math.min(1, (offset + READ_SIZE) / file.size))
  }
  onProgress?.(1)
  return hasher.digest('hex')
}

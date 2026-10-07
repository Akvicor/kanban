import {useSyncExternalStore} from 'react'
import {ApiError, errorMessage, ResultCode, type WriteResult} from '../api/client'
import {createFileAttachment} from '../api/file'
import type {Attachment} from '../api/types'
import {uploadFile, type UploadPhase} from './uploadFile'
import {t} from '../i18n'

/** 上传队列中的一项。phase 为 saving 时正在新建附件；error 不为空表示失败。 */
export interface UploadItem {
  id: number
  cardId: number
  name: string
  size: number
  phase: UploadPhase | 'waiting' | 'saving'
  ratio: number
  error: string
}

type Listener = () => void

/** 新建附件成功后的回调，用响应立即更新本地数据。 */
export type AttachmentCreated = (result: WriteResult<Attachment>) => void

interface Job {
  item: UploadItem
  file: File
  onCreated: AttachmentCreated
}

/**
 * 上传队列：多个文件依次上传，每个文件上传完成后新建附件。
 * 队列是全局的，关闭卡片详情后上传继续进行，重新打开时可以看到进度。失败的项留在队列中，直到用户移除。
 */
class UploadQueue {
  private items: UploadItem[] = []
  private jobs: Job[] = []
  private running = false
  private nextId = 1
  private readonly listeners = new Set<Listener>()

  subscribe = (listener: Listener) => {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  getItems = () => this.items

  /** 把文件加入队列，上传到 cardId 对应的卡片。 */
  add(cardId: number, files: File[], onCreated: AttachmentCreated) {
    for (const file of files) {
      const item: UploadItem = {id: this.nextId++, cardId, name: file.name, size: file.size, phase: 'waiting', ratio: 0, error: ''}
      this.items = [...this.items, item]
      this.jobs.push({item, file, onCreated})
    }
    this.emit()
    void this.run()
  }

  /** 移除一项（只能移除失败的项，进行中的上传不能取消）。 */
  dismiss(id: number) {
    this.items = this.items.filter((item) => item.id !== id || !item.error)
    this.emit()
  }

  private update(id: number, changes: Partial<UploadItem>) {
    this.items = this.items.map((item) => (item.id === id ? {...item, ...changes} : item))
    this.emit()
  }

  private emit() {
    this.listeners.forEach((listener) => listener())
  }

  private async run() {
    if (this.running) return
    this.running = true
    while (this.jobs.length > 0) {
      const job = this.jobs.shift()!
      try {
        await this.process(job)
        this.items = this.items.filter((item) => item.id !== job.item.id)
        this.emit()
      } catch (error) {
        this.update(job.item.id, {error: errorMessage(error)})
      }
    }
    this.running = false
  }

  private async process({item, file, onCreated}: Job) {
    // 新建附件时文件恰好被删除（冲突），重新上传一次。
    for (let attempt = 0; ; attempt++) {
      const sha = await uploadFile(file, ({phase, ratio}) => this.update(item.id, {phase, ratio}))
      this.update(item.id, {phase: 'saving', ratio: 1})
      try {
        onCreated(await createFileAttachment(item.cardId, sha, file.name.slice(0, 128) || t('upload.unnamed')))
        return
      } catch (error) {
        if (attempt > 0 || !(error instanceof ApiError) || error.code !== ResultCode.Conflict) throw error
      }
    }
  }
}

export const uploadQueue = new UploadQueue()

/** 某张卡片在上传队列中的项。 */
export function useUploads(cardId: number): UploadItem[] {
  const items = useSyncExternalStore(uploadQueue.subscribe, uploadQueue.getItems)
  return items.filter((item) => item.cardId === cardId)
}

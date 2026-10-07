import {
  DndContext,
  pointerWithin,
  useDraggable,
  useDroppable,
  type DragEndEvent,
  type DragMoveEvent,
} from '@dnd-kit/core'
import {useDragSensors} from '../../hooks/useDragSensors'
import {createContext, useCallback, useContext, useState, type ClipboardEvent, type FormEvent, type KeyboardEvent} from 'react'
import * as cardApi from '../../api/card'
import type {Task} from '../../api/types'
import {EntityType} from '../../sync/store'
import {MAX_TASK_DEPTH, resolveTaskDrop, taskProgress, taskTree, type TaskDropIntent, type TaskNode} from '../model'
import {useCardWrite} from './useCardWrite'
import {t, useT} from '../../i18n'

interface DropHint {
  overId: number
  intent: TaskDropIntent
}

const HintContext = createContext<DropHint | null>(null)

/** 开始拖动时指针的纵坐标，鼠标和触摸都适用。 */
function startY(event: Event): number {
  if ('touches' in event) {
    const touch = (event as TouchEvent).touches[0] ?? (event as TouchEvent).changedTouches[0]
    return touch?.clientY ?? 0
  }
  return (event as PointerEvent).clientY
}

/** 把粘贴或输入的文字拆成多行任务标题，空行忽略。 */
function splitLines(text: string): string[] {
  return text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)
}

/**
 * 任务：最多三层，勾选后标题变灰并加删除线，已完成的任务照常显示。
 * 可以拖动调整顺序或父任务：拖到任务上部或下部放在它前后，拖到中部成为它的子任务。
 * 在新建任务的输入框中粘贴多行文字时，每行创建一个任务。
 */
export function TaskSection({cardId, tasks, readOnly}: {cardId: number; tasks: Task[]; readOnly: boolean}) {
  const t = useT()
  const write = useCardWrite()
  const tree = taskTree(tasks, cardId)
  const progress = taskProgress(tasks, cardId)
  const [hint, setHint] = useState<DropHint | null>(null)
  const sensors = useDragSensors()
  const own = tasks.filter((task) => task.card_id === cardId)

  function onDragMove(event: DragMoveEvent) {
    if (!event.over) {
      setHint(null)
      return
    }
    const rect = event.over.rect
    const ratio = (startY(event.activatorEvent) + event.delta.y - rect.top) / rect.height
    setHint({overId: Number(event.over.id), intent: ratio < 0.3 ? 'before' : ratio > 0.7 ? 'after' : 'inside'})
  }

  function onDragEnd(event: DragEndEvent) {
    const current = hint
    setHint(null)
    if (!current || !event.over) return
    const draggedId = Number(event.active.id)
    const target = resolveTaskDrop(own, draggedId, current.overId, current.intent)
    if (target) {
      void write(EntityType.Task, () => cardApi.moveTask(draggedId, target.parentId, target.index))
    }
  }

  return (
    <section className="section">
      <h5>
        {t('task.section')}
        {progress.total > 0 && (
          <span className="extra">
            {progress.done}/{progress.total}
          </span>
        )}
      </h5>
      {progress.total > 0 && (
        <div className="progress" aria-hidden>
          <span style={{width: `${(progress.done / progress.total) * 100}%`}} />
        </div>
      )}
      <DndContext sensors={sensors} collisionDetection={pointerWithin} onDragMove={onDragMove} onDragOver={onDragMove} onDragEnd={onDragEnd} onDragCancel={() => setHint(null)}>
        <HintContext.Provider value={hint}>
          <TaskNodes nodes={tree} cardId={cardId} readOnly={readOnly} />
        </HintContext.Provider>
      </DndContext>
      {!readOnly && <TaskInput cardId={cardId} parentId={null} placeholder={t('task.addPlaceholder')} />}
    </section>
  )
}

function TaskNodes({nodes, cardId, readOnly}: {nodes: TaskNode[]; cardId: number; readOnly: boolean}) {
  if (nodes.length === 0) return null
  return (
    <ul className="tasks">
      {nodes.map((node) => (
        <TaskRow key={node.task.id} node={node} cardId={cardId} readOnly={readOnly} />
      ))}
    </ul>
  )
}

function TaskRow({node, cardId, readOnly}: {node: TaskNode; cardId: number; readOnly: boolean}) {
  const {task} = node
  const write = useCardWrite()
  const hint = useContext(HintContext)
  // 任务行内有勾选框和按钮，只挂指针和触摸的拖动监听，不加 dnd-kit 的 role="button" 等属性，避免交互控件嵌套。
  const {listeners, setNodeRef: setDragRef, isDragging} = useDraggable({id: task.id, disabled: readOnly})
  const {setNodeRef: setDropRef} = useDroppable({id: task.id, disabled: readOnly})
  const setRef = useCallback(
    (element: HTMLElement | null) => {
      setDragRef(element)
      setDropRef(element)
    },
    [setDragRef, setDropRef],
  )
  const [editing, setEditing] = useState<string | null>(null)
  const [adding, setAdding] = useState(false)

  let className = task.done ? 'task done' : 'task'
  if (isDragging) className += ' dragging'
  if (hint?.overId === task.id) className += ` drop-${hint.intent}`

  async function commit() {
    if (editing === null) return
    const title = editing.trim()
    setEditing(null)
    if (title && title !== task.title) {
      await write(EntityType.Task, () => cardApi.renameTask(task.id, title))
    }
  }

  return (
    <li>
      <div ref={setRef} className={className} {...(editing === null ? listeners : {})}>
        <button
          type="button"
          className="box"
          role="checkbox"
          aria-checked={task.done}
          aria-label={task.done ? t('task.markUndone') : t('common.done')}
          disabled={readOnly}
          onClick={() => void write(EntityType.Task, () => cardApi.setTaskDone(task.id, !task.done))}
        />
        {editing === null ? (
          <span className="t" onClick={() => !readOnly && setEditing(task.title)}>
            {task.title}
          </span>
        ) : (
          <input
            className="input task-input"
            value={editing}
            aria-label={t('task.title')}
            autoFocus
            onChange={(e) => setEditing(e.target.value)}
            onBlur={() => void commit()}
            onKeyDown={(e) => {
              if (e.key === 'Enter') e.currentTarget.blur()
              if (e.key === 'Escape') {
                e.stopPropagation()
                setEditing(null)
              }
            }}
          />
        )}
        {!readOnly && editing === null && (
          <span className="task-actions">
            {node.depth < MAX_TASK_DEPTH && (
              <button type="button" className="icon-btn sm" aria-label={t('task.addSubtask')} title={t('task.addSubtask')} onClick={() => setAdding(true)}>
                +
              </button>
            )}
            <button
              type="button"
              className="icon-btn sm"
              aria-label={t('task.delete')}
              title={t('task.deleteWithChildren')}
              onClick={() => void write(EntityType.Task, () => cardApi.deleteTask(task.id), task.id)}
            >
              ×
            </button>
          </span>
        )}
      </div>
      <TaskNodes nodes={node.children} cardId={cardId} readOnly={readOnly} />
      {adding && <TaskInput cardId={cardId} parentId={task.id} placeholder={t('task.subtasks')} onDone={() => setAdding(false)} />}
    </li>
  )
}

function TaskInput({cardId, parentId, placeholder, onDone}: {cardId: number; parentId: number | null; placeholder: string; onDone?: () => void}) {
  const write = useCardWrite()
  const [title, setTitle] = useState('')

  async function create(titles: string[]) {
    if (titles.length === 0) return
    if ((await write(EntityType.Task, () => cardApi.createTasks(cardId, parentId, titles))) === 'ok') {
      setTitle('')
    }
  }

  function onPaste(event: ClipboardEvent<HTMLInputElement>) {
    const lines = splitLines(event.clipboardData.getData('text'))
    if (lines.length > 1) {
      event.preventDefault()
      void create(lines)
    }
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Escape' && onDone) {
      event.stopPropagation()
      onDone()
    }
  }

  return (
    <form
      className={parentId === null ? 'task-new' : 'task-new nested'}
      onSubmit={(event: FormEvent) => {
        event.preventDefault()
        void create(splitLines(title))
      }}
    >
      <input
        className="input"
        value={title}
        placeholder={placeholder}
        aria-label={placeholder}
        autoFocus={parentId !== null}
        onChange={(e) => setTitle(e.target.value)}
        onPaste={onPaste}
        onKeyDown={onKeyDown}
        onBlur={() => title.trim() === '' && onDone?.()}
      />
    </form>
  )
}

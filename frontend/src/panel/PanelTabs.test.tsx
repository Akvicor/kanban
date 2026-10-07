import {fireEvent, render, screen} from '@testing-library/react'
import {MemoryRouter} from 'react-router-dom'
import {describe, expect, it, vi} from 'vitest'
import type {Panel} from '../api/types'
import {PanelTabs, type PanelTabActions} from './PanelTabs'

const panel: Panel = {id: 3, board_id: 1, name: '本季度', position: 1, archived_at: null}

function renderTabs() {
  const actions: PanelTabActions = {
    rename: vi.fn(),
    toggleMain: vi.fn(),
    options: vi.fn(),
    moveToBoard: vi.fn(),
    archive: vi.fn(),
    openListArchive: vi.fn(),
    openCardArchive: vi.fn(),
    reorder: vi.fn(),
    create: vi.fn(),
  }
  render(
    <MemoryRouter>
      <PanelTabs boardId={1} tabs={[panel]} activePanelId={3} mainPanelId={null} actions={actions} />
    </MemoryRouter>,
  )
  return actions
}

async function chooseMenu(name: string) {
  fireEvent.keyDown(screen.getByRole('button', {name: '本季度 的菜单'}), {key: 'Enter'})
  fireEvent.click(await screen.findByRole('menuitem', {name}))
}

describe('PanelTabs', () => {
  it('菜单中的列表归档打开该面板的列表归档', async () => {
    const actions = renderTabs()
    await chooseMenu('列表归档')
    expect(actions.openListArchive).toHaveBeenCalledWith(panel)
  })

  it('菜单中的卡片归档打开该面板的卡片归档', async () => {
    const actions = renderTabs()
    await chooseMenu('卡片归档')
    expect(actions.openCardArchive).toHaveBeenCalledWith(panel)
  })
})

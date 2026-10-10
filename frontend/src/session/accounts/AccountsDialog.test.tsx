import {fireEvent, render, screen, within} from '@testing-library/react'
import {MemoryRouter} from 'react-router-dom'
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest'
import {canLeavePage, logoutAccount, logoutAll, switchAccount} from './actions'
import {AccountsDialog} from './AccountsDialog'
import {ACCOUNTS_STORAGE_KEY, type AccountsState} from './storage'

vi.mock('./actions', () => ({
  canLeavePage: vi.fn(() => true),
  switchAccount: vi.fn(async () => {}),
  logoutAccount: vi.fn(async () => {}),
  logoutAll: vi.fn(async () => {}),
}))

const saved: AccountsState = {
  accounts: [
    {userId: 1, username: 'alice', nickname: '甲', token: 'alice-token'},
    {userId: 2, username: 'bob', nickname: '', token: 'bob-token'},
    {userId: 3, username: 'carol', nickname: '', token: null},
  ],
  current: 1,
}

function renderDialog(switchOnly = false) {
  render(
    <MemoryRouter>
      <AccountsDialog open onOpenChange={() => {}} switchOnly={switchOnly} />
    </MemoryRouter>,
  )
}

function row(name: string): HTMLElement {
  return screen.getByText(name, {selector: '.account-info .hint'}).closest('li')!
}

describe('AccountsDialog', () => {
  beforeEach(() => {
    localStorage.setItem(ACCOUNTS_STORAGE_KEY, JSON.stringify(saved))
    vi.mocked(canLeavePage).mockReturnValue(true)
    vi.clearAllMocks()
  })
  afterEach(() => localStorage.clear())

  it('列出账号：当前账号不能切换，已失效的账号显示重新登录', () => {
    renderDialog()
    expect(within(row('alice')).queryByRole('button', {name: '切换'})).toBeNull()
    expect(within(row('alice')).getByText('当前')).toBeInTheDocument()
    expect(within(row('carol')).getByRole('button', {name: '重新登录'})).toBeInTheDocument()

    fireEvent.click(within(row('bob')).getByRole('button', {name: '切换'}))
    expect(switchAccount).toHaveBeenCalledWith(2)
  })

  it('退出都需要二次确认', async () => {
    renderDialog()
    fireEvent.click(within(row('bob')).getByRole('button', {name: '退出'}))
    expect(logoutAccount).not.toHaveBeenCalled()
    fireEvent.click(within(screen.getByRole('dialog', {name: '退出账号 bob'})).getByRole('button', {name: '退出'}))
    expect(logoutAccount).toHaveBeenCalledWith(2)

    // 退出进行中按钮不可用，结束后再继续。
    const signOutAll = screen.getByRole('button', {name: '退出全部账号'})
    await vi.waitFor(() => expect(signOutAll).toBeEnabled())
    fireEvent.click(signOutAll)
    expect(logoutAll).not.toHaveBeenCalled()
    fireEvent.click(within(screen.getByRole('dialog', {name: '退出全部账号'})).getByRole('button', {name: '退出'}))
    expect(logoutAll).toHaveBeenCalledOnce()
  })

  it('有上传或请求进行中时不切换，提示稍后再试', () => {
    vi.mocked(canLeavePage).mockReturnValue(false)
    renderDialog()
    fireEvent.click(within(row('bob')).getByRole('button', {name: '切换'}))
    expect(switchAccount).not.toHaveBeenCalled()
    expect(screen.getByRole('alert')).toHaveTextContent('有正在进行的上传或请求')
  })

  it('只用于切换时不列出当前账号，也没有退出当前和全部账号', () => {
    renderDialog(true)
    expect(screen.queryByText('alice', {selector: '.account-info .hint'})).toBeNull()
    expect(screen.queryByRole('button', {name: '退出全部账号'})).toBeNull()
    expect(screen.queryByRole('button', {name: '登录新账号'})).toBeNull()
  })
})

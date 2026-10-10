/**
 * 账号的切换、添加和退出流程。会换账号的操作最后都刷新页面（见 reload.ts）。
 *
 * 文件 Cookie 每个站点只有一份，换账号时先用原账号的令牌注销它的文件令牌，再用目标账号的令牌签发新的，
 * 然后修改当前账号并刷新；页面打开后还会在后台签发一次作为兜底（见 SessionProvider）。
 * 注销或签发失败（例如令牌已失效）时照常继续，由刷新后的登录确认流程判断账号是否有效。
 */
import * as authApi from '../../api/auth'
import {pendingRequestCount} from '../../api/client'
import type {LoginResult} from '../../api/types'
import {uploadQueue} from '../../upload/queue'
import {reloadToHome} from './reload'
import {clearAccounts, currentToken, readAccounts, removeAccount, saveLogin, setCurrentAccount} from './storage'

/** 能否离开当前页面：没有正在上传的文件，也没有未结束的接口请求。切换和会刷新页面的退出都要先检查。 */
export function canLeavePage(): boolean {
  return pendingRequestCount() === 0 && !uploadQueue.isBusy()
}

/** 把文件 Cookie 从 fromToken 的设备换到 toToken 的设备，失败时忽略。 */
async function moveFileCookie(fromToken: string | null, toToken: string | null): Promise<void> {
  if (fromToken) await authApi.revokeFileCookie(fromToken).catch(() => {})
  if (toToken) await authApi.writeFileCookie(toToken).catch(() => {})
}

/** 切换到列表中的账号并刷新页面。目标账号已失效时同样切换，刷新后进入重新登录。 */
export async function switchAccount(userId: number): Promise<void> {
  const target = readAccounts().accounts.find((account) => account.userId === userId)
  if (!target) return
  await moveFileCookie(currentToken(), target.token)
  setCurrentAccount(userId)
  reloadToHome()
}

/**
 * 保存一次登录：账号已在列表中时原位置更新令牌，并吊销被替换的旧令牌（失败时忽略）。
 * 吊销旧令牌会清除浏览器中的文件 Cookie，调用方之后需要为当前账号重新签发。不修改当前账号。
 */
export async function rememberLogin(result: LoginResult): Promise<void> {
  const {replacedToken} = saveLogin(result.me.account, result.token)
  if (replacedToken) await authApi.logout(replacedToken).catch(() => {})
}

/** 已登录时登录另一个账号：加入列表（或原位置更新），切换过去并刷新页面。 */
export async function addAccount(result: LoginResult): Promise<void> {
  const fromToken = currentToken()
  await rememberLogin(result)
  // 重新登录的是当前账号时，原令牌已在 rememberLogin 中吊销，不再注销它的文件令牌。
  await moveFileCookie(fromToken === currentToken() ? fromToken : null, result.token)
  setCurrentAccount(result.me.account.id)
  reloadToHome()
}

/**
 * 退出一个账号：吊销其令牌（已失效的不请求服务端）并从列表移除，吊销失败时本地照常移除。
 * - 退出的是当前账号：切换到列表中的第一个账号并刷新页面，没有其他账号时刷新后进入登录页。
 * - 退出的是其他账号：当前账号不变，不刷新；登出请求会清除浏览器中的文件 Cookie（属于当前账号），之后为当前账号重新签发。
 */
export async function logoutAccount(userId: number): Promise<void> {
  const state = readAccounts()
  const account = state.accounts.find((item) => item.userId === userId)
  if (!account) return
  if (account.token) await authApi.logout(account.token).catch(() => {})
  removeAccount(userId)

  if (userId !== state.current) {
    if (account.token && currentToken()) await authApi.writeFileCookie().catch(() => {})
    return
  }
  const next = readAccounts().accounts[0] ?? null
  await moveFileCookie(null, next?.token ?? null)
  setCurrentAccount(next?.userId ?? null)
  reloadToHome()
}

/** 退出全部账号：逐个吊销令牌，清空列表并刷新页面，刷新后进入登录页。 */
export async function logoutAll(): Promise<void> {
  for (const account of readAccounts().accounts) {
    if (account.token) await authApi.logout(account.token).catch(() => {})
  }
  clearAccounts()
  reloadToHome()
}

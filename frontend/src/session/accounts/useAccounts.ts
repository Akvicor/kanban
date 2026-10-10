import {useSyncExternalStore} from 'react'
import {readAccounts, subscribeAccounts, type AccountsState} from './storage'

/** 本设备保存的账号列表和当前账号，随本标签页和其他标签页的修改更新。 */
export function useAccounts(): AccountsState {
  return useSyncExternalStore(subscribeAccounts, readAccounts)
}

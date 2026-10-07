/** 界面上显示的名字：有昵称用昵称，否则用用户名。 */
export function displayName(account: {username: string; nickname?: string}): string {
  return account.nickname || account.username
}

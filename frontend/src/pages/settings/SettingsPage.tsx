import {MenuButton} from '../../layout/MenuButton'
import {AppearanceSection} from './AppearanceSection'
import {DeviceSection} from './DeviceSection'
import {GeneralSection} from './GeneralSection'
import {PasswordSection} from './PasswordSection'
import {ProfileSection} from './ProfileSection'
import {ShortcutSection} from './ShortcutSection'
import {TemplateSection} from './TemplateSection'
import './SettingsPage.css'
import {useT} from '../../i18n'

/** 个人设置。每一项单独保存，保存后同步到该用户的其他设备。 */
export function SettingsPage() {
  const t = useT()
  return (
    <div className="page settings-page">
      <div className="page-header">
        <MenuButton />
        <h1>{t('nav.settings')}</h1>
      </div>
      <ProfileSection />
      <AppearanceSection />
      <GeneralSection />
      <TemplateSection />
      <ShortcutSection />
      <PasswordSection />
      <DeviceSection />
    </div>
  )
}

import {useMemo} from 'react'
import {Select} from '../../components/Select'
import {useT} from '../../i18n'
import {useSettings} from '../../sync/hooks'
import {timeZoneOptions} from '../../utils/datetime'
import {useSaveSettings} from './useSaveSettings'

/** 语言、时区和进入主页时的行为。修改后立即保存。 */
export function GeneralSection() {
  const t = useT()
  const settings = useSettings()
  const {save, saving, error} = useSaveSettings()
  const zones = useMemo(() => timeZoneOptions(settings.timezone), [settings.timezone])

  return (
    <section className="card-section">
      <h2>{t('settings.general')}</h2>
      <p className="hint">{t('settings.generalHint')}</p>
      <div className="form">
        <label className="field">
          <span>{t('settings.language')}</span>
          <Select
            label={t('settings.language')}
            value={settings.locale}
            disabled={saving}
            options={[
              {value: '', label: t('common.followSystem')},
              {value: 'zh-CN', label: '简体中文'},
              {value: 'en', label: 'English'},
            ]}
            onChange={(value) => void save({locale: value})}
          />
        </label>
        <label className="field">
          <span>{t('settings.timezone')}</span>
          <Select
            label={t('settings.timezone')}
            value={settings.timezone}
            disabled={saving}
            options={zones.map((zone) => ({value: zone, label: zone}))}
            onChange={(value) => void save({timezone: value})}
          />
        </label>
        <label className="switch">
          <input
            type="checkbox"
            checked={settings.open_main_board_on_home}
            disabled={saving}
            onChange={(e) => void save({open_main_board_on_home: e.target.checked})}
          />
          <span className="track" />
          {t('settings.openMainBoardOnHome')}
        </label>
        {error && <p className="form-error">{error}</p>}
      </div>
    </section>
  )
}

import type {PalettePreference} from '../../theme/palette'
import {useSettings} from '../../sync/hooks'
import {useSaveSettings} from './useSaveSettings'
import {t, useT} from '../../i18n'

interface PaletteOption {
  value: PalettePreference
  name: string
  description: string
  /** 预览色块：背景、卡片、强调色。跟随系统时左右各显示亮色和暗色。 */
  swatch: string[]
}

/** 配色选项。文案随界面语言变化，必须在渲染时构造，不能做成模块级常量。 */
function paletteOptions(): PaletteOption[] {
  return [
    {value: 'system', name: t('common.followSystem'), description: t('settings.paletteSystem'), swatch: ['#F7F8FA', '#0C66E4', '#0E1014', '#7C9CFF']},
    {value: 'clean', name: t('settings.paletteClean'), description: t('settings.paletteCleanDesc'), swatch: ['#F7F8FA', '#FFFFFF', '#0C66E4']},
    {value: 'dark', name: t('settings.paletteDark'), description: t('settings.paletteDarkDesc'), swatch: ['#0E1014', '#1A1E28', '#7C9CFF']},
    {value: 'paper', name: t('settings.palettePaper'), description: t('settings.palettePaperDesc'), swatch: ['#F4EFE6', '#FFFDF8', '#C2552B']},
  ]
}

/** 配色选择。选择后立即保存并应用。 */
export function AppearanceSection() {
  const t = useT()
  const settings = useSettings()
  const {save, saving, error} = useSaveSettings()
  const current = settings.palette
  const options = paletteOptions()

  return (
    <section className="card-section">
      <h2>{t('settings.palette')}</h2>
      <p className="hint">{t('settings.paletteHint')}</p>
      <div className="palette-options" role="radiogroup" aria-label={t('settings.palette')}>
        {options.map((option) => (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={current === option.value}
            className={current === option.value ? 'palette-option selected' : 'palette-option'}
            disabled={saving}
            onClick={() => current !== option.value && void save({palette: option.value})}
          >
            <span className="palette-swatch">
              {option.swatch.map((color, index) => (
                <span key={index} style={{background: color}} />
              ))}
            </span>
            <strong>{option.name}</strong>
            <span className="hint">{option.description}</span>
          </button>
        ))}
      </div>
      {error && <p className="form-error">{error}</p>}
    </section>
  )
}

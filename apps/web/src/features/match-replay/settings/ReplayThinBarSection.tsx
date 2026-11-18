/**
 * ReplayThinBarSection — L'OPTION « BARRE FINE » du panneau de la vue : un OUI/NON qui garde, ou non, la barre de
 * lecture fine et translucide au Plein cadre (`THIN_BAR_KEY`). Allumée, la barre est posée sur le bas du plan et ne
 * prend sa forme pleine que quand on s'en sert ; éteinte, elle reste pleine, sous le plan.
 *
 * La section n'existe qu'au Plein cadre (`available`) : ailleurs la barre est fixe et l'interrupteur n'aurait
 * aucun effet à l'écran.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { SettingsToggle } from './ReplaySettingsToggle'

/** Ce que le panneau sait de la barre fine : si le cadre peut la poser, si elle est voulue, et le geste. */
export interface ReplayThinBarControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

export function ThinBarSection({ locale, thinBar }: { locale: ReplayLocale; thinBar: ReplayThinBarControls }) {
  const t = REPLAY_TEXT[locale].frames
  if (!thinBar.available) return null
  return (
    <section className="space-y-1">
      <SettingsToggle label={t.thinBar} pressed={thinBar.show} onToggle={thinBar.onToggle} hint={t.thinBarHint} />
    </section>
  )
}

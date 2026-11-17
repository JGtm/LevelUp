/**
 * ReplayPlanFeedSection — L'OPTION « FIL DES ÉLIMINATIONS » du panneau de la vue : un OUI/NON qui pose, ou non,
 * le fil des cinq dernières éliminations sur le plan (`ReplayFeedOverlay`).
 *
 * La section n'existe que là où le fil du plan peut se poser (`available` : Cinéma et Plein cadre, rangée de fiches
 * posée) : un interrupteur sans effet à l'écran tromperait. Au Standard, et dans le repli où la colonne porte
 * le fil complet, elle ne se rend pas.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { SettingsToggle } from './ReplaySettingsToggle'

/** Ce que le panneau sait du fil du plan : si le cadre peut le poser, s'il l'est, et le geste. */
export interface ReplayPlanFeedControls {
  available: boolean
  show: boolean
  onToggle: () => void
}

export function PlanFeedSection({ locale, planFeed }: { locale: ReplayLocale; planFeed: ReplayPlanFeedControls }) {
  const t = REPLAY_TEXT[locale].frames
  if (!planFeed.available) return null
  return (
    <section className="space-y-1">
      <SettingsToggle label={t.planFeed} pressed={planFeed.show} onToggle={planFeed.onToggle} hint={t.planFeedHint} />
    </section>
  )
}

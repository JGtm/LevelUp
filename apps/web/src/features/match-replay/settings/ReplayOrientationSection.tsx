/**
 * ReplayOrientationSection — LA COMMANDE D'ORIENTATION DE LA CARTE dans le panneau de la vue.
 *
 * Un CHOIX EXCLUSIF à quatre positions (0°, 90°, 180°, 270° — quarts de tour antihoraires), donc un
 * segmenté (`SettingsSegments`, `role="radiogroup"`) et pas quatre interrupteurs : exactement une
 * position est vraie. Les degrés ne portent aucun mot : ils s'écrivent pareil dans toutes les
 * langues, seul le titre est traduit.
 *
 * Le réglage est retenu par carte (`useReplayOrientation`) ; la section n'existe que quand la
 * carte est connue (`available`).
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { normalizeTurns, QUARTER_TURNS, turnsToDegrees, type QuarterTurns } from '../model/orientation'
import { SettingsSegments } from './ReplaySettingsToggle'

/** Ce que le panneau sait de l'orientation : la position, le geste, et si la carte la porte. */
export interface ReplayOrientationControls {
  available: boolean
  turns: QuarterTurns
  onSelect: (turns: QuarterTurns) => void
}

/** Les positions, telles que la commande les affiche : « 0° », « 90° », « 180° », « 270° ». */
const OPTIONS = QUARTER_TURNS.map((turns) => ({ value: String(turns), label: `${turnsToDegrees(turns)}°` }))

export function OrientationSection({
  locale, orientation,
}: {
  locale: ReplayLocale
  orientation: ReplayOrientationControls
}) {
  const t = REPLAY_TEXT[locale]
  if (!orientation.available) return null
  return (
    <section className="space-y-1">
      <SettingsSegments
        label={t.orientationTitle}
        value={String(orientation.turns)}
        options={OPTIONS.map((o) => ({ ...o, hint: t.orientationHint }))}
        onSelect={(value) => orientation.onSelect(normalizeTurns(Number(value)))}
      />
    </section>
  )
}

/**
 * ReplayOrientationSection — LA COMMANDE D'ORIENTATION DE LA CARTE dans le panneau de la vue.
 *
 * Un CHOIX EXCLUSIF à cinq positions — « Auto », puis 0°, 90°, 180°, 270° (quarts de tour horaires) —,
 * donc un segmenté (`SettingsSegments`, `role="radiogroup"`) et pas cinq interrupteurs : exactement une
 * position est vraie. Les degrés ne portent aucun mot : ils s'écrivent pareil dans toutes les
 * langues, seuls le titre et « Auto » sont traduits.
 *
 * « Auto » est la position tant qu'aucun choix n'est retenu : le plan se tourne tout seul quand il y gagne
 * de la place (`autoTurns`). La choisir EFFACE le choix retenu de la carte ; choisir un quart de tour le
 * retient (`useReplayOrientation`). La section n'existe que quand la carte est connue (`available`).
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { normalizeTurns, QUARTER_TURNS, turnsToDegrees, type QuarterTurns } from '@/lib/replay/orientation'
import { SettingsSegments } from './ReplaySettingsToggle'

/** Ce que le panneau sait de l'orientation : le choix retenu (`null` = Auto), le geste, et si la carte la porte. */
export interface ReplayOrientationControls {
  available: boolean
  retained: QuarterTurns | null
  onSelect: (turns: QuarterTurns | null) => void
}

/** La valeur de la position « Auto » : aucune des quatre valeurs de quarts de tour. */
const AUTO = 'auto'

/** Les positions chiffrées, telles que la commande les affiche : « 0° », « 90° », « 180° », « 270° ». */
const OPTIONS = QUARTER_TURNS.map((turns) => ({ value: String(turns), label: `${turnsToDegrees(turns)}°` }))

export function OrientationSection({
  locale, orientation,
}: {
  locale: ReplayLocale
  orientation: ReplayOrientationControls
}) {
  const t = REPLAY_TEXT[locale]
  if (!orientation.available) return null
  const options = [
    { value: AUTO, label: t.frames.orientationAuto, hint: t.frames.orientationAutoHint },
    ...OPTIONS.map((o) => ({ ...o, hint: t.orientationHint })),
  ]
  return (
    <section className="space-y-1">
      <SettingsSegments
        label={t.orientationTitle}
        value={orientation.retained === null ? AUTO : String(orientation.retained)}
        options={options}
        onSelect={(value) => orientation.onSelect(value === AUTO ? null : normalizeTurns(Number(value)))}
      />
    </section>
  )
}

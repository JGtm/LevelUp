/**
 * TacticalPiluleLecture — la pilule « Lecture » du bandeau du plan : les sept lectures de placement et, quand
 * le titre déclare `spawn_forecast`, « Réapparitions adverses ». Partagée par la carte du plan et par celle
 * de la lecture des réapparitions (une seule pilule, une seule liste).
 */
import type { TacticalText } from './i18n'
import { ETIQUETTE, PILULE } from './pilules'
import { LECTURE_REAPPARITIONS, type LectureDuPlan } from './reapparitions.logic'

export function PiluleLecture({
  t,
  valeur,
  onChange,
  onIntent,
  avecReapparitions,
}: {
  t: TacticalText
  valeur: LectureDuPlan
  onChange: (lecture: LectureDuPlan) => void
  /** La pilule est survolée ou prend le focus : l'intention de changer de lecture. */
  onIntent?: () => void
  /** « Réapparitions adverses » est proposée (capability `spawn_forecast` du titre). */
  avecReapparitions: boolean
}) {
  return (
    <label className={PILULE} onPointerEnter={onIntent} onFocus={onIntent}>
      <span className={ETIQUETTE}>{t.pillReading}</span>
      <select
        value={valeur}
        onChange={(e) => onChange(e.target.value as LectureDuPlan)}
        aria-label={t.pillReading}
        className="w-[146px] bg-transparent pl-1.5 pr-0.5 text-foreground"
      >
        {t.analysisQuestions.map((q) => (
          <option key={q.id} value={q.id}>
            {q.label}
          </option>
        ))}
        {avecReapparitions && <option value={LECTURE_REAPPARITIONS}>{t.reappQuestion}</option>}
      </select>
    </label>
  )
}

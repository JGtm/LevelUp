/**
 * ViesEnPilule — la bascule « Départ / Toutes les vies » du bandeau du plan : quelles apparitions du
 * joueur forment les grappes de réapparition (cf. `vies.logic`). Même forme que la pilule « Joueurs ».
 */
import type { TacticalText } from './i18n'
import { ETIQUETTE, PILULE } from './pilules'
import { VIES_VALEURS, type TacticalVies } from './vies.logic'

export function ViesEnPilule({
  t,
  vies,
  onViesChange,
}: {
  t: TacticalText
  vies: TacticalVies
  onViesChange: (vies: TacticalVies) => void
}) {
  const libelle: Record<TacticalVies, string> = { depart: t.livesStart, toutes: t.livesAll }
  return (
    <span role="group" aria-label={t.pillLives} title={t.livesHint} className={PILULE} data-testid="tactical-plan-vies">
      <span className={ETIQUETTE}>{t.pillLives}</span>
      {VIES_VALEURS.map((valeur, i) => (
        <button
          key={valeur}
          type="button"
          aria-pressed={vies === valeur}
          onClick={() => onViesChange(valeur)}
          className={`px-[7px] text-foreground aria-pressed:bg-primary aria-pressed:font-medium aria-pressed:text-primary-foreground${i > 0 ? ' border-l border-input' : ''}`}
        >
          {libelle[valeur]}
        </button>
      ))}
    </span>
  )
}

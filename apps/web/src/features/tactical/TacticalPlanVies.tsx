/**
 * ViesEnPilule — la bascule « Départ / Toutes les vies » du bandeau du plan : quelles apparitions du
 * joueur forment les grappes de réapparition (cf. `vies.logic`). Même forme que la pilule « Joueurs ».
 */
import type { TacticalGrappe } from '@/lib/api/types'
import { useCapability } from '@/lib/capabilities/capabilities'

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

/**
 * PilulesDeReapparition — le filtre de grappe (« Réapparition ») et la bascule « Vies ».
 *
 * Le filtre n'a rien à choisir sans amas ; un filtre déjà posé reste visible pour pouvoir l'ôter. La
 * bascule ne dépend pas des grappes servies (sous « Départ » une carte peut n'en avoir aucune alors que
 * « Toutes les vies » en aurait) : elle est offerte dès que le titre déclare `spawn_forecast`, et jamais
 * sans (le serveur refuserait `toutes`). La capability est lue par le chemin du bootstrap
 * (`useCapability`), jamais par le slug du titre.
 */
export function PilulesDeReapparition({
  t,
  spawn,
  onSpawnChange,
  grappes,
  vies,
  onViesChange,
}: {
  t: TacticalText
  spawn: string
  onSpawnChange: (spawn: string) => void
  grappes: readonly TacticalGrappe[]
  vies: TacticalVies
  onViesChange: (vies: TacticalVies) => void
}) {
  const spawnForecast = useCapability('spawn_forecast')
  return (
    <>
      {(grappes.length > 0 || spawn !== '') && (
        <label className={PILULE}>
          <span className={ETIQUETTE}>{t.pillRespawn}</span>
          <select
            value={spawn}
            onChange={(e) => onSpawnChange(e.target.value)}
            aria-label={t.pillRespawn}
            className="w-[72px] bg-transparent pl-1.5 pr-0.5 text-foreground"
          >
            <option value="">{t.pillRespawnAll}</option>
            {grappes.map((g) => (
              <option key={g.id} value={g.id}>
                {g.nom}
              </option>
            ))}
          </select>
        </label>
      )}
      {spawnForecast && <ViesEnPilule t={t} vies={vies} onViesChange={onViesChange} />}
    </>
  )
}

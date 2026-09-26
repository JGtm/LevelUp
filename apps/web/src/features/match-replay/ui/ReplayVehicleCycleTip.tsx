/**
 * ReplayVehicleCycleTip — L'INFOBULLE D'UN EMPLACEMENT DE NAISSANCE DE VÉHICULE, au survol de son
 * losange.
 *
 * TROIS LIGNES AU PLUS, ET CHACUNE RÉPOND À UNE QUESTION : ce qui naît ici (la famille dominante,
 * nommée par le document ; le titre générique quand la table des châssis ne nomme rien), quand
 * ça revient (le compte à rebours, ou « un véhicule s'y trouve »), et sur quoi la prédiction
 * repose (la médiane avec ses déciles, et le nombre de cycles mesurés).
 *
 * POURQUOI LA RÉSERVE EST ICI ET PAS SUR LA CARTE — c'est le verdict du 2026-08-18 sur les socles
 * d'arme, repris tel quel : la médiane, les déciles et le nombre d'écarts disent la CONFIANCE
 * dans le cycle, ce qui est une lecture d'analyse, pas un repère de carte. Le losange, lui, ne
 * porte que le lieu et le compte.
 *
 * LES DEUX COMPTES À REBOURS SE DISTINGUENT COMME CHEZ LES SOCLES (D3, 2026-08-27) : celui qui
 * vise la prochaine naissance VUE DANS LE FILM est exact et se dit tel quel, celui que le CYCLE
 * prédit garde son « ≈ ». Les libellés sont LES MÊMES (`padRespawnMeasuredFmt` /
 * `padRespawnExpectedFmt`) : c'est la même question, et deux phrasés pour une même réserve se
 * seraient contredits à la première retouche.
 *
 * PAS DE SOURCE = PAS DE LIGNE. Ni naissance suivante, ni fin datée d'où partir : ni chiffre, ni
 * tiret — un tiret suggérerait qu'on saurait.
 *
 * Purement présentationnel : l'occupation et le compte à rebours sont calculés au survol
 * (`useReplayVehicles`), la géométrie dans `vehicleCyclesLayer`.
 */
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { VehicleCycleHover } from '../layers/useReplayVehicles'

/** Décalage de l'infobulle sous le pointeur, en pixels (même valeur que celle des socles). */
const TIP_OFFSET = 12
/** Largeur estimée : elle sert UNIQUEMENT à décider du côté, jamais à contraindre le rendu. */
const TIP_WIDTH = 192

interface ReplayVehicleCycleTipProps {
  locale: ReplayLocale
  hover: VehicleCycleHover
  /** Largeur du canvas : elle borne l'infobulle du côté droit. */
  width: number
}

export function ReplayVehicleCycleTip({ locale, hover, width }: ReplayVehicleCycleTipProps) {
  const t = REPLAY_TEXT[locale]
  const { at, name, occupied, respawn, cycle } = hover
  const respawnText = respawn
    ? (respawn.measured ? t.padRespawnMeasuredFmt : t.padRespawnExpectedFmt)(respawn.seconds)
    : occupied
      ? t.vehicleCycleOccupied
      : null
  const flip = at.x + TIP_OFFSET + TIP_WIDTH > width
  return (
    <div
      role="tooltip"
      className="pointer-events-none absolute z-10 max-w-[13rem] rounded border border-border bg-card px-2 py-1 text-xs shadow-lg"
      style={{
        left: flip ? undefined : at.x + TIP_OFFSET,
        right: flip ? Math.max(width - at.x + TIP_OFFSET, 0) : undefined,
        top: at.y + TIP_OFFSET,
      }}
    >
      <span className="block font-medium">{name}</span>
      {respawnText !== null && <span className="block text-muted-foreground">{respawnText}</span>}
      <span className="block text-muted-foreground">
        {t.vehicleCycleFmt(cycle.medianS, cycle.p10S, cycle.p90S)} ·{' '}
        {t.vehicleCycleGapsFmt(cycle.gaps)}
      </span>
    </div>
  )
}

/**
 * sessionPerMinute.logic — les cadences par minute d'UNE session (frags, morts, assistances,
 * dégâts infligés et subis) et les bornes d'axe communes à deux sessions comparées.
 *
 * Lu par le graphe « Stats par minute » (`SessionFdaBars`) ET par l'échelle partagée de la
 * comparaison (`perMinuteDomains`) : une seule définition de la cadence, donc l'axe figé de la
 * comparaison ne peut pas diverger de ce que le graphe dessine.
 *
 * Règles (mêmes que le groupe « Rythme » de Tendances, `analysis/trends/indicators_pace.go`) :
 * rapport de la SOMME des comptes à la SOMME des minutes jouées ; un match sans durée est exclu ;
 * un dégât inconnu exclut le match de CE dégât (numérateur et dénominateur) ; sans aucun match
 * retenu, la cadence est absente (`null`), jamais zéro.
 */
import type { SessionDetailMatchRow } from '@/lib/api/types'
import { alignedZeroAxes } from '@/lib/charts/alignedZeroAxes'
import { pooledPerMinute } from '@/lib/charts/perMinute'

export interface SessionPerMinuteRates {
  frags: number
  deaths: number
  assists: number
  /** Absent (`null`) sans aucun match à dégâts infligés connus. */
  damageDealt: number | null
  /** Absent (`null`) sans dégâts subis connus, ou quand le titre ne les fournit pas. */
  damageTaken: number | null
}

/** Bornes des deux axes du graphe : comptes à gauche, dégâts à droite (zéro commun). */
export interface PerMinuteDomains {
  fdaMinute: [number, number]
  /** Absent quand aucune des deux sessions n'a de dégâts. */
  damageMinute?: [number, number]
}

const durationOf = (m: SessionDetailMatchRow) => m.duration_seconds

/**
 * Cadences d'une session ; `null` quand aucun match n'a de durée (rien à rapporter).
 * `withTaken` faux (titre sans dégâts subis, Halo 5) : les dégâts subis sont absents même si une
 * valeur nulle est rangée.
 */
export function sessionPerMinuteRates(
  matches: readonly SessionDetailMatchRow[],
  withTaken: boolean,
): SessionPerMinuteRates | null {
  const frags = pooledPerMinute(matches, (m) => m.kills, durationOf)
  const deaths = pooledPerMinute(matches, (m) => m.deaths, durationOf)
  const assists = pooledPerMinute(matches, (m) => m.assists, durationOf)
  if (frags == null || deaths == null || assists == null) return null
  return {
    frags,
    deaths,
    assists,
    damageDealt: pooledPerMinute(matches, (m) => m.damage_dealt, durationOf),
    damageTaken: withTaken ? pooledPerMinute(matches, (m) => m.damage_taken, durationOf) : null,
  }
}

/** Étendues (sous / au-dessus du zéro) des deux axes pour un lot de sessions. */
function extents(rates: readonly SessionPerMinuteRates[]) {
  const dealt = rates.map((r) => r.damageDealt ?? 0)
  const taken = rates.map((r) => r.damageTaken ?? 0)
  return {
    left: {
      neg: Math.max(...rates.map((r) => r.deaths), 0),
      pos: Math.max(...rates.flatMap((r) => [r.frags, r.assists]), 0),
    },
    right: { neg: Math.max(...taken, 0), pos: Math.max(...dealt, 0) },
    hasDamage: rates.some((r) => r.damageDealt != null || r.damageTaken != null),
  }
}

/**
 * Bornes d'axe des cadences, pour UNE session (zéros alignés) ou pour deux sessions comparées
 * (les plus larges des deux). Sans dégâts, l'axe de gauche garde ses bornes `[-morts, +max]` et il
 * n'y a pas d'axe de droite. `undefined` quand aucune session n'a de cadence.
 */
export function perMinuteDomains(
  sessions: readonly (readonly SessionDetailMatchRow[])[],
  withTaken: boolean,
): PerMinuteDomains | undefined {
  const rates = sessions
    .map((matches) => sessionPerMinuteRates(matches, withTaken))
    .filter((r): r is SessionPerMinuteRates => r != null)
  if (rates.length === 0) return undefined
  const { left, right, hasDamage } = extents(rates)
  if (!hasDamage) return { fdaMinute: [-left.neg, left.pos] }
  const aligned = alignedZeroAxes(left, right)
  return { fdaMinute: aligned.left, damageMinute: aligned.right }
}

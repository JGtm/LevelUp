/**
 * timeseriesPerMinute.logic — cadences par minute de CHAQUE match du graphe « Stats par minute »
 * de Séries temporelles : frags, morts, assistances, dégâts infligés et subis.
 *
 * Un match sans temps joué n'a aucune cadence ; un dégât inconnu (`null`) reste absent, jamais zéro
 * (un point manquant, pas un point à zéro). Les dégâts subis sont écartés pour un titre qui ne les
 * fournit pas (Halo 5), même quand une valeur nulle est rangée.
 */
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { perMinute } from '@/lib/charts/perMinute'

export interface MatchPace {
  kills: number | null
  deaths: number | null
  assists: number | null
  damageDealt: number | null
  damageTaken: number | null
}

/** Cadences d'un match (`null` = pas de cadence : durée ou compte inconnu). */
export function matchPace(row: TimeseriesMatchRow, withTaken: boolean): MatchPace {
  const secs = row.time_played_seconds
  const rate = (count: number | null | undefined) => (count == null ? null : perMinute(count, secs))
  return {
    kills: rate(row.kills),
    deaths: rate(row.deaths),
    assists: rate(row.assists),
    damageDealt: rate(row.damage_dealt),
    damageTaken: withTaken ? rate(row.damage_taken) : null,
  }
}

/** Au moins un match porte une cadence de dégâts (sinon le graphe n'a ni courbe ni axe de droite). */
export function paceHasDamage(rows: readonly TimeseriesMatchRow[], withTaken: boolean): boolean {
  return rows.some((r) => {
    const p = matchPace(r, withTaken)
    return p.damageDealt != null || p.damageTaken != null
  })
}

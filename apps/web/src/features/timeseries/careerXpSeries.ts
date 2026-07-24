/**
 * careerXpSeries — construction pure (testable) des séries du chart « XP de
 * carrière (estimée) ». Séparé du composant ECharts pour tester la logique de
 * cumul sans monter de canvas (cf. reference_echarts_jsdom_test_mock).
 *
 * Entrée : les lignes par match (TimeseriesMatchRow), dont `career_xp_estimated`
 * est renseigné par le backend uniquement pour les titres portant la capability
 * analytics.career_xp_estimate ET les matchs PvP à score connu (nul sinon).
 */
import type { TimeseriesMatchRow } from '@/lib/api/types'

export interface CareerXpSeries {
  /** Vrai si au moins un match porte une estimation (sinon : chart masqué). */
  hasData: boolean
  /** XP estimée gagnée par match (null = match sans estimation → barre omise). */
  perMatch: (number | null)[]
  /**
   * XP de carrière cumulée : reportée sur les matchs sans estimation (pas de
   * retour à zéro) ; null tant qu'aucune XP connue (évite un plateau à 0 en tête).
   */
  cumulative: (number | null)[]
}

export function buildCareerXpSeries(rows: TimeseriesMatchRow[]): CareerXpSeries {
  const perMatch: (number | null)[] = rows.map((r) => r.career_xp_estimated ?? null)
  let running = 0
  let started = false
  const cumulative: (number | null)[] = rows.map((r) => {
    if (r.career_xp_estimated != null) {
      running += r.career_xp_estimated
      started = true
    }
    return started ? running : null
  })
  return {
    hasData: rows.some((r) => r.career_xp_estimated != null),
    perMatch,
    cumulative,
  }
}

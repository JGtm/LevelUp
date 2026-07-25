/**
 * careerXpSeries — construction pure (testable) des séries du chart « XP de
 * carrière (estimée) ». PARTAGÉ entre Timeseries (TimeseriesCareerXP) et Sessions
 * (SessionCareerXP, V72-13) : type d'entrée STRUCTUREL minimal — seul
 * `career_xp_estimated` importe pour ce builder, quel que soit le type de ligne
 * réel (TimeseriesMatchRow / SessionDetailMatchRow) — pour éviter toute
 * dépendance croisée feature→feature entre les deux pages consommatrices.
 *
 * `career_xp_estimated` est renseigné par le backend uniquement pour les titres
 * portant la capability analytics.career_xp_estimate ET les matchs PvP à score
 * connu (nul sinon).
 */

/** Type structurel minimal — n'importe quelle ligne de match portant ce champ. */
export interface CareerXpRow {
  career_xp_estimated?: number | null
}

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

export function buildCareerXpSeries(rows: CareerXpRow[]): CareerXpSeries {
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

/**
 * useActiveSeason — détecte la saison qui matche pile une fenêtre `period`.
 *
 * Hook pur dérivé : pas de queryKey, pas d'effet. La saison "active" est
 * un état purement dérivé de la `period` courante du store global filtres
 * croisée avec le catalog de saisons (useSeasons).
 *
 * Comportement :
 *   - Si `period.start_date` ou `period.end_date` est null → null
 *   - Si la fenêtre matche pile une saison (égalité ISO YYYY-MM-DD) → cette saison
 *   - Sinon → null (cas d'un preset 7j/30j/90j ou d'une plage custom)
 */

import { useMemo } from 'react'

import type { PeriodInput } from '@/lib/api/types'
import { useSeasons, type SeasonEntry } from '@/lib/i18n/fieldMappings'
import { findActiveSeason, isoDateUTC } from '@/lib/seasons/findSeasonAt'

/** Convertit une saison vers son couple ISO `YYYY-MM-DD` exploitable par
 *  setPeriod. Pour une saison ouverte, end_date = aujourd'hui en UTC. */
export function seasonToPeriod(s: SeasonEntry): PeriodInput {
  return {
    start_date: isoDateUTC(s.startDate),
    end_date: isoDateUTC(s.endDate ?? new Date()),
  }
}

export function useActiveSeason(period: PeriodInput | undefined): {
  seasons: SeasonEntry[]
  activeSeason: SeasonEntry | null
} {
  const seasons = useSeasons()
  const activeSeason = useMemo(
    () => findActiveSeason(seasons, period?.start_date, period?.end_date),
    [seasons, period],
  )
  return { seasons, activeSeason }
}

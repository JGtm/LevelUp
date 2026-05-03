/**
 * Helpers purs de résolution chronologique sur la liste des saisons.
 *
 * Aucune dépendance React → testables sans renderer. Utilisés par
 * useActiveSeason (squad) et par les boutons prev/next du PeriodSessionRail
 * en mode "season".
 */

import type { SeasonEntry } from '@/lib/i18n/fieldMappings'

/**
 * Retourne la saison qui couvre l'instant `at`, ou null si aucune.
 *
 * Convention de fenêtre : [startDate, endDate). Une saison ouverte
 * (endDate === null) couvre tout instant >= startDate.
 */
export function findSeasonAt(seasons: SeasonEntry[], at: Date): SeasonEntry | null {
  for (const s of seasons) {
    if (at < s.startDate) continue
    if (s.endDate === null || at < s.endDate) return s
  }
  return null
}

/** Retourne la saison qui couvre l'instant présent (ou null). */
export function currentSeason(seasons: SeasonEntry[]): SeasonEntry | null {
  return findSeasonAt(seasons, new Date())
}

/** Voisine chronologique précédente (par displayOrder), ou null si la
 *  saison passée est la première. */
export function prevSeason(seasons: SeasonEntry[], current: SeasonEntry): SeasonEntry | null {
  const idx = seasons.findIndex((s) => s.id === current.id)
  return idx > 0 ? seasons[idx - 1] : null
}

/** Voisine chronologique suivante (par displayOrder), ou null si la
 *  saison passée est la dernière. */
export function nextSeason(seasons: SeasonEntry[], current: SeasonEntry): SeasonEntry | null {
  const idx = seasons.findIndex((s) => s.id === current.id)
  return idx >= 0 && idx < seasons.length - 1 ? seasons[idx + 1] : null
}

/** Convertit une Date UTC vers son ISO `YYYY-MM-DD` (pour matcher les
 *  PeriodInput stockés en strings ISO côté store). */
export function isoDateUTC(d: Date): string {
  const yyyy = d.getUTCFullYear()
  const mm = String(d.getUTCMonth() + 1).padStart(2, '0')
  const dd = String(d.getUTCDate()).padStart(2, '0')
  return `${yyyy}-${mm}-${dd}`
}

/** Détecte si une fenêtre `(start, end)` ISO matche pile une saison du
 *  catalog. Pour une saison ouverte, end est comparé à `today` (UTC)
 *  car c'est ce que setPeriod injecte par convention.
 *
 *  Retourne null si :
 *    - start ou end vaut null/undefined
 *    - aucune saison ne matche pile
 */
export function findActiveSeason(
  seasons: SeasonEntry[],
  startISO: string | null | undefined,
  endISO: string | null | undefined,
): SeasonEntry | null {
  if (!startISO || !endISO) return null
  const todayISO = isoDateUTC(new Date())
  for (const s of seasons) {
    if (isoDateUTC(s.startDate) !== startISO) continue
    const expectedEnd = isoDateUTC(s.endDate ?? new Date(todayISO))
    if (expectedEnd === endISO) return s
  }
  return null
}

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

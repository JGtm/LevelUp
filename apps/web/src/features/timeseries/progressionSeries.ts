/**
 * Construction des séries de progression CSR/LUSR — transformation pure extraite
 * de TimeseriesSkillProgression.tsx pour que le module de composant n'exporte que
 * des composants (react-refresh/only-export-components).
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import { lusrChainLabel } from '@/features/career/lusr-chains'
import type { TimeseriesMatchRow } from '@/lib/api/types'

export interface ProgressionSeries {
  key: string
  label: string
  group: string
  ratingType: string
  /** Valeurs de classement indexées par position de match (null hors matchs notés). */
  values: (number | null)[]
  /** Points de placement : [indexMatch, y]. */
  placementPoints: Array<[number, number]>
  /** Index de match où débute une nouvelle saison (rupture de courbe). */
  seasonBreaks: number[]
}

function groupKey(row: TimeseriesMatchRow): string {
  return `${row.skill_rating_type ?? ''}:${row.skill_playlist_group ?? ''}`
}

export function buildProgressionSeries(rows: TimeseriesMatchRow[], locale: ManifestLocale): ProgressionSeries[] {
  const n = rows.length

  // Regrouper les matchs notés par clé (ratingType:playlistGroup), en conservant
  // leur index de position (= index de catégorie X, aligné sur buildMatchCategories).
  const byKey = new Map<string, Array<{ row: TimeseriesMatchRow; idx: number }>>()
  rows.forEach((r, idx) => {
    if (r.skill_rating_value == null) return
    const k = groupKey(r)
    if (!byKey.has(k)) byKey.set(k, [])
    byKey.get(k)!.push({ row: r, idx })
  })

  const result: ProgressionSeries[] = []

  for (const [, entries] of byKey) {
    const first = entries[0].row
    const ratingType = first.skill_rating_type ?? ''
    const group = first.skill_playlist_group ?? ''
    const label = `${lusrChainLabel(group, locale)} (${ratingType.toUpperCase()})`

    // Midpoint y pour les diamants de placement : milieu de la plage des valeurs réelles.
    // Pour CSR, la valeur stockée pendant les placements est 0.0 — afficher à 0 serait trompeur.
    const realValues = entries
      .filter(e => (e.row.skill_measurement_remaining ?? 0) === 0 && e.row.skill_rating_value != null)
      .map(e => e.row.skill_rating_value as number)
    const placementY = realValues.length > 0
      ? (Math.min(...realValues) + Math.max(...realValues)) / 2
      : null

    const values = new Array<number | null>(n).fill(null)
    const placementPoints: Array<[number, number]> = []
    const seasonBreaks: number[] = []
    let prevSeasonId: string | null | undefined

    for (const { row, idx } of entries) {
      const isPlacement = (row.skill_measurement_remaining ?? 0) > 0

      if (isPlacement) {
        // Placements : y = milieu de la plage des valeurs réelles (pas 0.0 du CSR en DB).
        if (placementY !== null) placementPoints.push([idx, placementY])
        continue // values[idx] reste null → coupure de ligne
      }

      // Rupture de saison : season_id différent du précédent match noté.
      if (prevSeasonId != null && row.skill_season_id && prevSeasonId !== row.skill_season_id) {
        seasonBreaks.push(idx)
      }

      values[idx] = row.skill_rating_value!
      prevSeasonId = row.skill_season_id ?? prevSeasonId
    }

    result.push({
      key: `skill.${ratingType}.${group}`,
      label,
      group,
      ratingType,
      values,
      placementPoints,
      seasonBreaks,
    })
  }

  return result
}

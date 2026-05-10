/**
 * matchLabels (timeseries) — wrappers spécifiques aux charts par-match de la
 * page timeseries. Le helper pur `truncateMap` vit désormais dans
 * `@/lib/charts/matchLabels` (consommé aussi par squad + engagement).
 *
 * Format aligné sur SquadPerformanceCharts (page Contributions Squad) :
 *   `#N\nMap` (2 lignes), `Map` tronquée à 9 caractères avec coupure
 *   intelligente sur ' ' ou '-'.
 */
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { truncateMap } from '@/lib/charts/matchLabels'

export { truncateMap }

/** Construit les étiquettes X au format `#N\nMap` (ou `#N` si pas de carte). */
export function buildMatchCategories(rows: TimeseriesMatchRow[]): string[] {
  return rows.map((r, i) => {
    const map = r.map_name_fr || r.map_name
    return map ? `#${i + 1}\n${truncateMap(map)}` : `#${i + 1}`
  })
}

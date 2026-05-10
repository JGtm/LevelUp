// Construction des séries pour le chart "Évolution LUSR / CSR".
// Extrait de CareerChartsSection.tsx pour testabilité (Fast Refresh impose
// qu'un .tsx n'exporte que des composants).
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { CareerLusrCheckpoint } from '@/lib/api/types'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'

export type LusrSeriesMeta = {
  label: string
  groupKey: string
  ratingType: string
  [k: string]: unknown
}

// Les 4 groupes officiels du calcul LUSR (cf. apps/go-api/internal/sync/skill_config.go).
// `social` est de la donnée historique : alias d'`arena` pour l'affichage.
const GROUP_LABEL_KEYS = {
  arena: 'career.lusr.group.arena',
  btb: 'career.lusr.group.btb',
  fun: 'career.lusr.group.fun',
  ranked: 'career.lusr.group.ranked',
} as const

export type CanonicalGroup = keyof typeof GROUP_LABEL_KEYS

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

export function isCanonicalGroup(g: string): g is CanonicalGroup {
  return g in GROUP_LABEL_KEYS
}

// `social` (legacy) → `arena`. Toute valeur inconnue → `arena` (default Go).
export function normalizeGroup(raw: string | null | undefined): CanonicalGroup {
  const g = (raw ?? '').trim().toLowerCase()
  if (g === 'social' || g === '') return 'arena'
  return isCanonicalGroup(g) ? g : 'arena'
}

export function lusrGroupLabel(group: CanonicalGroup, locale: ManifestLocale): string {
  return careerManifest[GROUP_LABEL_KEYS[group]][locale]
}

export function buildLusrSeries(
  checkpoints: CareerLusrCheckpoint[],
  locale: ManifestLocale,
): ChartSeries<[string, number]>[] {
  // Une courbe par (rating_type, playlist_group). 4 groupes max : arena/btb/fun/ranked.
  // Le calcul LUSR maintient un état mu/sigma par playlist_group (cf.
  // skill_rating_loaders.go::loadExistingLUSRStates) — afficher par groupe
  // reflète exactement la sémantique du calcul.
  //
  // Filtre : checkpoints dont playlist_name est un UUID brut (pas de résolution
  // dans asset_translations) sont écartés. La donnée reste en DB.
  // Sur même date pour le même groupe, le checkpoint le plus récent (tri ASC
  // côté Go) écrase.
  const byKey = new Map<string, { group: CanonicalGroup; ratingType: string; pts: Map<string, number> }>()

  for (const cp of checkpoints) {
    if (!cp.recorded_at) continue
    const name = (cp.playlist_name ?? '').trim()
    if (UUID_RE.test(name)) continue

    const group = normalizeGroup(cp.playlist_group)
    const ratingType = cp.rating_type ?? 'LUSR'
    const seriesKey = `${ratingType}:${group}`
    const date = cp.recorded_at.slice(0, 10)

    if (!byKey.has(seriesKey)) {
      byKey.set(seriesKey, { group, ratingType, pts: new Map() })
    }
    byKey.get(seriesKey)!.pts.set(date, cp.rating_value)
  }

  return Array.from(byKey.entries()).map(([seriesKey, { group, ratingType, pts }]) => {
    const label = `${lusrGroupLabel(group, locale)} (${ratingType})`
    const meta: LusrSeriesMeta = { label, groupKey: group, ratingType }
    return {
      key: `career.lusr.${seriesKey}`,
      meta,
      datapoints: Array.from(pts.entries())
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([date, val]) => [date, val] as [string, number]),
    }
  })
}

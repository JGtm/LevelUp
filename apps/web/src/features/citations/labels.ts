/**
 * labels.ts — résolution locale-aware des CLÉS STABLES de catégorie émises par
 * le backend Citations vers leurs libellés FR/EN.
 *
 * Title-agnostic : depuis V7.3 lot 2 (item 1.4), l'API ne sert plus de libellé
 * humain (« MULTIPLAYER » côté Halo 5, « Mode de jeu » côté Infinite) mais une
 * clé canonique (`multiplayer`, `game_mode`, `weapon`, `vehicle`, `enemy`,
 * `spartan_companies`, `other` — cf. internal/games/canonical/
 * commendation_category.go). La localisation vit ici, dans citations.toml.
 *
 * Une clé absente du manifeste retombe sur la CLÉ BRUTE : dégradation visible
 * mais non cassante, strictement le même contrat que features/medals/labels.ts.
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import { citationsManifest } from '@/lib/i18n/generated/citations'

type ManifestEntry = Readonly<Record<ManifestLocale, string>>
const MANIFEST = citationsManifest as Readonly<Record<string, ManifestEntry>>

export function citationCategoryLabel(key: string, locale: ManifestLocale): string {
  const entry = MANIFEST[`citations.category.${key}`]
  return entry ? entry[locale] : key
}

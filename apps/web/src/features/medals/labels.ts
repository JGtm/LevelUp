/**
 * labels.ts — résolution locale-aware des CLÉS STABLES émises par le backend
 * Médailles (catégorie, super-section, rareté) vers leurs libellés FR/EN.
 *
 * Title-agnostic : le backend n'envoie que des clés (ex. "ctf", "game_modes",
 * "heroic") ; la localisation vit ici. Une clé absente du manifeste (medal_type
 * d'un futur titre non encore traduit) retombe sur la CLÉ BRUTE — dégradation
 * visible mais non cassante, cohérente avec formatMessage().
 *
 * Les libellés n'ont pas de placeholder ICU → accès direct au manifeste (pas de
 * MessageFormat). Les templates avec placeholders (en-tête, aria) restent servis
 * par formatMessage() côté composant avec des clés statiques typées.
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import { medalsManifest } from '@/lib/i18n/generated/medals'

type ManifestEntry = Readonly<Record<ManifestLocale, string>>
const MANIFEST = medalsManifest as Readonly<Record<string, ManifestEntry>>

/** Clés de rareté Halo Infinite (index 0..3). Halo 5 utilise une difficulty_key
 *  NUMÉRIQUE ("1", "2"…) → hors de cet ensemble → pas de pastille de rareté. */
const RARITY_KEYS: ReadonlySet<string> = new Set(['normal', 'heroic', 'legendary', 'mythic'])

function resolve(prefix: string, key: string, locale: ManifestLocale): string {
  const entry = MANIFEST[`medals.${prefix}.${key}`]
  return entry ? entry[locale] : key
}

export function medalCategoryLabel(key: string, locale: ManifestLocale): string {
  return resolve('category', key, locale)
}

export function medalSuperSectionLabel(key: string, locale: ManifestLocale): string {
  return resolve('super_section', key, locale)
}

/**
 * Libellé de rareté à afficher, ou `null` si la clé n'est pas une rareté Halo
 * Infinite connue (dont les difficulty_key numériques de Halo 5) — le composant
 * n'affiche alors AUCUNE pastille de rareté.
 */
export function medalRarityLabel(difficultyKey: string, locale: ManifestLocale): string | null {
  if (!RARITY_KEYS.has(difficultyKey)) return null
  return resolve('rarity', difficultyKey, locale)
}

/**
 * staticAssets.ts — composition d'URLs d'assets statiques title-scopés côté frontend.
 *
 * Pendant Phase 6 du plan finition multi-titres, le format des URLs `/static/...`
 * bascule de **flat** (`/static/ranks/X.png`) vers **title-scoped**
 * (`/static/ranks/{titleSlug}/X.png`). Le flip est synchronisé avec la migration
 * FS atomique côté Go (Phase 6.5) via le flag transitionnel
 * `VITE_STATIC_PATHS_TITLE_SCOPED`.
 *
 * Les composants React doivent utiliser ce module au lieu de hardcoder
 * `/static/...` dans des `src=`.
 *
 * Symétrique au backend Go : `internal/assets/static/` (couche 2 pure) +
 * `internal/games/halo_infinite/adapter_asset_urls.go` (couche 3, flag
 * `STATIC_PATHS_TITLE_SCOPED`).
 */

// Default depuis Phase 6.5 : title-scoped activé. Set ENV à "false" pour rollback.
const TITLE_SCOPED = import.meta.env.VITE_STATIC_PATHS_TITLE_SCOPED !== 'false'

const MOUNT_POINT = '/static'

const FOLDER: Record<StaticKind, string> = {
  map: 'maps',
  medal: 'medals/icons',
  'csr-rank': 'ranks',
  weapon: 'weapons-assets',
  commendation: 'commendations',
}

export type StaticKind = 'map' | 'medal' | 'csr-rank' | 'weapon' | 'commendation'

export const DEFAULT_TITLE_SLUG = 'halo_infinite'

/**
 * staticAssetURL compose l'URL d'un asset statique selon le flag
 * VITE_STATIC_PATHS_TITLE_SCOPED.
 *
 * @param kind type d'asset (sous-dossier sous /static/)
 * @param id   identifiant pré-encodé (path-safe, sans extension)
 * @param ext  extension fichier (avec point, ex: ".png")
 * @param titleSlug slug du titre (default "halo_infinite")
 *
 * Format : /static/{folder}/{titleSlug}/{id}{ext} (title-scoped)
 *      ou : /static/{folder}/{id}{ext} (flat)
 */
export function staticAssetURL(
  kind: StaticKind,
  id: string,
  ext: string,
  titleSlug: string = DEFAULT_TITLE_SLUG,
): string {
  if (!id) return ''
  if (TITLE_SCOPED) {
    return `${MOUNT_POINT}/${FOLDER[kind]}/${titleSlug}/${id}${ext}`
  }
  return `${MOUNT_POINT}/${FOLDER[kind]}/${id}${ext}`
}

/**
 * csrRankImageURL retourne l'URL du badge CSR pour un tier + sub-tier.
 * Cas spécial Onyx : passer subTier=0 ou tier="Onyx" (pas de sub-tier).
 */
export function csrRankImageURL(tier: string, subTier: number, titleSlug?: string): string {
  if (!tier) return ''
  if (tier === 'Onyx' || subTier <= 0) {
    return staticAssetURL('csr-rank', '120px-HINF-CSR_Onyx', '.png', titleSlug)
  }
  return staticAssetURL('csr-rank', `120px-HINF-CSR_${tier}${subTier}`, '.png', titleSlug)
}

/**
 * unrankedBadgeURL retourne l'URL du badge "Unranked" (cas joueurs sans rang).
 */
export function unrankedBadgeURL(titleSlug?: string): string {
  return staticAssetURL('csr-rank', 'Unranked', '.png', titleSlug)
}

/**
 * staticAssets.ts — composition d'URLs d'assets statiques title-scopés côté frontend.
 *
 * Format émis : `/static/{folder}/{titleSlug}/{id}{ext}` (title-scoped) en
 * cohérence avec l'arborescence FS et les URLs émises par le backend Go.
 *
 * Les composants React doivent utiliser ce module au lieu de hardcoder
 * `/static/...` dans des `src=`.
 *
 * Symétrique au backend Go : `internal/assets/static/` (couche 2 pure) +
 * `internal/games/halo_infinite/adapter_asset_urls.go` (couche 3).
 */

const MOUNT_POINT = '/static'

const FOLDER: Record<StaticKind, string> = {
  map: 'maps',
  medal: 'medals',
  'csr-rank': 'ranks',
  weapon: 'weapons-assets',
  commendation: 'commendations',
}

export type StaticKind = 'map' | 'medal' | 'csr-rank' | 'weapon' | 'commendation'

export const DEFAULT_TITLE_SLUG = 'halo_infinite'

/**
 * staticAssetURL compose l'URL d'un asset statique title-scopé.
 *
 * @param kind type d'asset (sous-dossier sous /static/)
 * @param id   identifiant pré-encodé (path-safe, sans extension)
 * @param ext  extension fichier (avec point, ex: ".png")
 * @param titleSlug slug du titre (default "halo_infinite")
 *
 * Format : /static/{folder}/{titleSlug}/{id}{ext}.
 */
export function staticAssetURL(
  kind: StaticKind,
  id: string,
  ext: string,
  titleSlug: string = DEFAULT_TITLE_SLUG,
): string {
  if (!id) return ''
  return `${MOUNT_POINT}/${FOLDER[kind]}/${titleSlug}/${id}${ext}`
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
 * unrankedBadgeURL retourne l'URL du badge unranked_0.png (0 placement effectué).
 */
export function unrankedBadgeURL(titleSlug?: string): string {
  return staticAssetURL('csr-rank', 'unranked_0', '.png', titleSlug)
}

/**
 * unrankedBadgeURLForProgress retourne l'URL du badge unranked_N.png proportionnel
 * à une progression de placement (done/total). Miroir CÔTÉ FRONT de la formule
 * backend `unrankedBadgeURLForThreshold` (apps/go-api/internal/platform/duckdb/
 * home_repo_skill_peak.go) : N = done*10/total, clampé [0,9] — les 10 images
 * disponibles (unranked_0..9) sont réutilisées quel que soit le seuil réel
 * (5 pour CSR depuis Season 3, 10 pour LUSR/CSR legacy).
 *
 * `total` <= 0 est un garde-fou (donnée incohérente) : retombe sur unranked_0.
 */
export function unrankedBadgeURLForProgress(done: number, total: number, titleSlug?: string): string {
  if (total <= 0) return unrankedBadgeURL(titleSlug)
  let n = Math.floor((done * 10) / total)
  if (n < 0) n = 0
  if (n > 9) n = 9
  return staticAssetURL('csr-rank', `unranked_${n}`, '.png', titleSlug)
}

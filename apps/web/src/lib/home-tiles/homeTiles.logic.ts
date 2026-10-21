/**
 * Logique pure des tuiles de match de l'Accueil (D-8) : la tête est servie avec `/pages/home`, la
 * suite par pages de `/pages/home/matches` chaînées par curseur, les favoris par la même route en
 * `kind=favorites`. Aucun React ici : testable sans DOM.
 */
import type { HomeMatchesPage, HomePageResponse, RecentMatchItem } from '@/lib/api/types'

/** Taille d'une page demandée (le serveur sert 30 par défaut, 60 au plus). */
export const HOME_TILES_PAGE_SIZE = 30

/** Les champs de la tête qui décrivent la liste de tuiles. */
export type HomeTilesHead = Pick<
  HomePageResponse,
  'recent_matches' | 'recent_matches_has_more' | 'recent_matches_next_before'
>

export interface HomeTiles {
  /** Tête puis pages chargées consécutives, sans doublon, dans l'ordre d'affichage. */
  tiles: RecentMatchItem[]
  /** Des tuiles plus anciennes existent après la dernière chargée. */
  hasMore: boolean
  /** Curseur de la page à demander ensuite ; absent sans suite. */
  nextBefore: string | undefined
}

/**
 * Assemble la tête et les pages. `pages[i]` est la page demandée avec le i-ème curseur ; une page
 * pas encore chargée (`undefined`) arrête l'assemblage : on n'affiche jamais une tuile au-delà d'un
 * trou. Un match déjà vu est ignoré (une même tuile ne peut pas s'afficher deux fois).
 */
export function mergeHomeTiles(head: HomeTilesHead, pages: ReadonlyArray<HomeMatchesPage | undefined>): HomeTiles {
  const seen = new Set<string>()
  const tiles: RecentMatchItem[] = []
  const push = (items: ReadonlyArray<RecentMatchItem>) => {
    for (const it of items) {
      if (seen.has(it.match_id)) continue
      seen.add(it.match_id)
      tiles.push(it)
    }
  }
  push(head.recent_matches ?? [])
  let hasMore = head.recent_matches_has_more === true
  let nextBefore = head.recent_matches_next_before
  for (const page of pages) {
    if (!page) break
    push(page.items ?? [])
    hasMore = page.has_more === true
    nextBefore = page.next_before
  }
  const more = hasMore && !!nextBefore
  return { tiles, hasMore: more, nextBefore: more ? nextBefore : undefined }
}

/** URL d'une page de tuiles : la suite après `before`, ou les favoris. */
export function homeMatchesUrl(playerSlug: string, params: { before?: string; favorites?: boolean }): string {
  const q = new URLSearchParams()
  if (params.favorites) q.set('kind', 'favorites')
  else {
    if (params.before) q.set('before', params.before)
    q.set('limit', String(HOME_TILES_PAGE_SIZE))
  }
  return `/players/${playerSlug}/pages/home/matches?${q.toString()}`
}

/**
 * Pose le drapeau favori d'un match dans une réponse de l'Accueil déjà en cache (tête, page de
 * tuiles ou favoris). Rend la MÊME référence quand rien ne change (pas de rendu inutile) et ignore
 * toute forme inconnue.
 */
export function patchFavoriteFlag<T>(data: T, matchId: string, favorite: boolean): T {
  if (!data || typeof data !== 'object') return data
  const holder = data as { recent_matches?: RecentMatchItem[]; items?: RecentMatchItem[] }
  const key = Array.isArray(holder.recent_matches) ? 'recent_matches' : Array.isArray(holder.items) ? 'items' : null
  if (!key) return data
  const list = holder[key] as RecentMatchItem[]
  if (!list.some((m) => m.match_id === matchId && m.is_favorite !== favorite)) return data
  return {
    ...data,
    [key]: list.map((m) => (m.match_id === matchId ? { ...m, is_favorite: favorite } : m)),
  }
}

/**
 * navContext.ts — Contexte de navigation chaînée entre matchs.
 *
 * Phase 2a (rework header MatchView, 2026-05-05) :
 *   - `MatchNavContext` décrit la liste de matchs adjacents et l'origine du
 *     filtrage. Quand l'utilisateur ouvre un match depuis une page filtrée
 *     (history, session, explorer...), on capte cette liste pour que les
 *     boutons prev/next sur la page match restent dans le même périmètre.
 *
 * Cascade de résolution dans `useMatchNeighbors` :
 *   1. Router state (instantané, scope onglet courant)
 *   2. sessionStorage (survit F5 / nav arrière)
 *   3. (Phase 2b) URL query params — survit Ctrl+Click + lien partagé
 *   4. fallback API global Q25
 *
 * Pas d'import de TanStack Router ici : module pur, testable sans DOM.
 * `localStorage` côté navigateur est tolérant aux erreurs (mode privé,
 * quota dépassé) — toutes les opérations sont fail-open silencieuses.
 */

/**
 * `filterSpec` est une interface placeholder qui sera remplie en Phase 2b
 * pour acheminer les filtres de la page source jusqu'à l'endpoint
 * `/neighbors?...`. Phase 2a utilise uniquement `matchIds` + `filtersLabel`.
 */
export interface MatchFilterSpec {
  playlist_name?: string
  mode_category?: string
  date_from?: string // ISO 8601
  date_to?: string // ISO 8601
  session_id?: string
  outcome?: 'win' | 'loss' | 'draw' | 'dnf'
}

export type MatchNavSource =
  | 'home_recent'
  | 'home_favorites'
  | 'history'
  | 'session'
  | 'citation'
  | 'media'

export interface MatchNavContext {
  source: MatchNavSource
  /** Liste ordonnée chronologique DESC (récent en tête) — index 0 = match courant le plus récent. */
  matchIds: string[]
  /** Texte humain pré-localisé. Affiché tel quel sous le compteur. */
  filtersLabel?: string
  /** Pré-rempli pour Phase 2b — filtres canoniques pour le fallback API. */
  filterSpec?: MatchFilterSpec
}

const STORAGE_PREFIX = 'levelup:matchNav:'
/** TTL : 1h. Au-delà, on retombe sur le fallback Q25 global. */
const TTL_MS = 60 * 60 * 1000

interface PersistedContext {
  ctx: MatchNavContext
  ts: number
}

/**
 * Sauvegarde le contexte de navigation pour le `matchId` cible.
 *
 * Fail-open : si `sessionStorage` jette (mode privé Safari, quota dépassé,
 * etc.), on ignore — l'utilisateur retombera sur le fallback API au F5.
 */
export function persistNavContext(matchId: string, ctx: MatchNavContext): void {
  if (!matchId || !ctx?.matchIds?.length) return
  try {
    const payload: PersistedContext = { ctx, ts: Date.now() }
    sessionStorage.setItem(STORAGE_PREFIX + matchId, JSON.stringify(payload))
  } catch {
    // ignore (mode privé, quota, sessionStorage absent)
  }
}

/**
 * Lit le contexte sauvegardé pour `matchId` ou retourne `null`.
 *
 * Si l'entrée a expiré (> TTL_MS), elle est purgée et `null` est retourné.
 */
export function readNavContext(matchId: string): MatchNavContext | null {
  if (!matchId) return null
  let raw: string | null = null
  try {
    raw = sessionStorage.getItem(STORAGE_PREFIX + matchId)
  } catch {
    return null
  }
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw) as PersistedContext
    if (
      typeof parsed?.ts !== 'number' ||
      !parsed.ctx ||
      !Array.isArray(parsed.ctx.matchIds)
    ) {
      return null
    }
    if (Date.now() - parsed.ts > TTL_MS) {
      try {
        sessionStorage.removeItem(STORAGE_PREFIX + matchId)
      } catch {
        // ignore
      }
      return null
    }
    return parsed.ctx
  } catch {
    return null
  }
}

/**
 * Purge l'entrée pour `matchId`. Utilisé par "↩ Sortir du contexte"
 * dans la barre de navigation.
 */
export function clearNavContext(matchId: string): void {
  if (!matchId) return
  try {
    sessionStorage.removeItem(STORAGE_PREFIX + matchId)
  } catch {
    // ignore
  }
}

/**
 * Calcule la position du match courant dans la liste et les voisins
 * adjacents. Pure — testable sans DOM ni router.
 *
 * @returns null si le match n'est pas dans la liste (cas anormal, fallback API).
 */
export function resolveNeighborsFromContext(
  ctx: MatchNavContext,
  matchId: string,
): {
  prev_match_id: string | null
  next_match_id: string | null
  current_index: number
  total: number
} | null {
  const idx = ctx.matchIds.indexOf(matchId)
  if (idx < 0) return null
  return {
    // matchIds est trié DESC (le plus récent en tête) — donc :
    //   "match précédent dans le temps" = index suivant dans le tableau (idx + 1)
    //   "match suivant dans le temps"   = index précédent dans le tableau (idx - 1)
    // Cette sémantique est miroir de Q25 backend (cf. queries_match.go).
    next_match_id: idx > 0 ? ctx.matchIds[idx - 1] ?? null : null,
    prev_match_id: idx < ctx.matchIds.length - 1 ? ctx.matchIds[idx + 1] ?? null : null,
    current_index: idx,
    total: ctx.matchIds.length,
  }
}

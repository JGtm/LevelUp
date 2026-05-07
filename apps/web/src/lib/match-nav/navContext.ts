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
 * `filterSpec` — miroir TS du `domain.MatchFilterSpec` Go (Phase 2b).
 * Sérialisé en query params pour le fallback API (?playlist=...&mode=...).
 *
 * Toutes les chaînes sont validées côté backend (whitelist + regex) — un
 * filtre invalide est ignoré (log warn, jamais 400). Le front sérialise
 * tel quel sans validation préalable.
 */
export type MatchFilterOutcome = 'win' | 'loss' | 'draw' | 'dnf'

export interface MatchFilterSpec {
  playlist_name?: string
  mode_category?: string
  date_from?: string // ISO 8601 RFC3339 ("2026-04-01T00:00:00Z")
  date_to?: string
  session_id?: string
  outcome?: MatchFilterOutcome
  /** XUID du joueur (coéquipier) avec qui le match a été joué — Phase 2c. */
  with_player_xuid?: string
}

/**
 * Sérialise un MatchFilterSpec en query string (sans le `?` initial).
 *
 * Mapping vers les noms de query params attendus par le handler Go :
 *   playlist_name    → playlist
 *   mode_category    → mode
 *   date_from        → from
 *   date_to          → to
 *   session_id       → session
 *   outcome          → outcome
 *   with_player_xuid → with_player
 *
 * Retourne une chaîne vide si la spec est vide ou nulle.
 */
export function filterSpecToQueryString(spec: MatchFilterSpec | null | undefined): string {
  if (!spec) return ''
  const params = new URLSearchParams()
  if (spec.playlist_name) params.set('playlist', spec.playlist_name)
  if (spec.mode_category) params.set('mode', spec.mode_category)
  if (spec.date_from) params.set('from', spec.date_from)
  if (spec.date_to) params.set('to', spec.date_to)
  if (spec.session_id) params.set('session', spec.session_id)
  if (spec.outcome) params.set('outcome', spec.outcome)
  if (spec.with_player_xuid) params.set('with_player', spec.with_player_xuid)
  return params.toString()
}

/**
 * Parse un MatchFilterSpec depuis un objet `search` (TanStack Router) ou
 * URLSearchParams. Retourne null si rien de pertinent.
 */
export function parseFilterSpecFromSearch(
  search: Record<string, unknown> | URLSearchParams | null | undefined,
): MatchFilterSpec | null {
  if (!search) return null
  const get = (key: string): string | undefined => {
    if (search instanceof URLSearchParams) {
      return search.get(key) ?? undefined
    }
    const v = (search as Record<string, unknown>)[key]
    return typeof v === 'string' ? v : undefined
  }

  const spec: MatchFilterSpec = {}
  const playlist = get('playlist')
  if (playlist) spec.playlist_name = playlist
  const mode = get('mode')
  if (mode) spec.mode_category = mode
  const from = get('from')
  if (from) spec.date_from = from
  const to = get('to')
  if (to) spec.date_to = to
  const session = get('session')
  if (session) spec.session_id = session
  const outcome = get('outcome')
  if (outcome === 'win' || outcome === 'loss' || outcome === 'draw' || outcome === 'dnf') {
    spec.outcome = outcome
  }
  // with_player : XUID numérique uniquement (validation backend identique).
  // Format Halo XUID : entier décimal jusqu'à 32 chars (ex: 2533274791785593).
  const withPlayer = get('with_player')
  if (withPlayer && /^\d{1,32}$/.test(withPlayer)) {
    spec.with_player_xuid = withPlayer
  }

  if (
    !spec.playlist_name &&
    !spec.mode_category &&
    !spec.date_from &&
    !spec.date_to &&
    !spec.session_id &&
    !spec.outcome &&
    !spec.with_player_xuid
  ) {
    return null
  }
  return spec
}

export type MatchNavSource =
  | 'home_recent'
  | 'home_favorites'
  | 'history'
  | 'session'
  | 'citation'
  | 'media'

/**
 * `ContextDescriptor` — Phase 2c : description sémantique typée du contexte
 * de navigation, utilisée pour construire un label compact dans la nav bar
 * du match (`Matchs <ctx> X/Y`).
 *
 * Préféré à `filtersLabel` (chaîne libre) car :
 *   - typé : aucune variante non couverte par i18n
 *   - localisable côté target (le label est construit dans la locale du match)
 *   - sérialisable / réplicable depuis n'importe quelle source (home, squad, …)
 *
 * Sources alimentant chaque variante :
 *   - `recent` / `favorites`   : HomePage tuile match, Synthesis highlights
 *   - `media`                  : MediaPage galerie + viewers
 *   - `with_player`            : Squad v2 history table (focus coéquipier)
 *   - `session`                : SessionDetailPage / Squad session table
 *   - `period`                 : Explorer / MatchHistory avec filtre date
 *   - `playlist` / `mode`      : Explorer / MatchHistory avec filtre simple
 *   - `top_matches`            : Career top matches table
 */
export type ContextDescriptor =
  | { kind: 'recent' }
  | { kind: 'favorites' }
  | { kind: 'media' }
  | { kind: 'with_player'; gamertag: string }
  | { kind: 'session'; startTimeUtc: string }
  | { kind: 'period'; from?: string; to?: string }
  | { kind: 'playlist'; name: string }
  | { kind: 'mode'; category: string }
  | { kind: 'top_matches' }

export interface MatchNavContext {
  source: MatchNavSource
  /** Liste ordonnée chronologique DESC (récent en tête) — index 0 = match courant le plus récent. */
  matchIds: string[]
  /**
   * Descriptor typé du contexte d'origine — Phase 2c.
   * Préféré à `filtersLabel` ; le builder `buildDescriptorLabel` côté
   * `match-view/i18n.ts` produit un label localisé compact.
   */
  contextDescriptor?: ContextDescriptor
  /**
   * Texte humain pré-localisé (legacy Phase 2a).
   * Conservé pour compat ; ignoré si `contextDescriptor` est fourni.
   */
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

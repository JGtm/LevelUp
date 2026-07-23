/**
 * Helper E2E — construction centralisée des URLs de ROUTE FRONT.
 *
 * POURQUOI. Depuis le chantier « titre dans l'URL » (feat/title-slug-in-url,
 * .ai/PLAN_TITLE_SLUG_URL_2026-07.md), les routes joueur vivent sous
 * `/t/{titleSlug}/players/{playerSlug}/…` : le titre actif est un SEGMENT d'URL,
 * plus un état de session implicite. Les specs tournent contre la session serveur
 * dont le titre par défaut est `halo_infinite` (cf. bootstrap). Centraliser la
 * construction d'URL (règle CLAUDE.md « ≤ 2 copies ») évite de recopier le préfixe
 * `/t/halo_infinite/players/…` dans chaque `page.goto` et fait qu'une évolution du
 * schéma d'URL ne se corrige qu'ICI.
 *
 * PORTÉE. Ce helper ne concerne QUE les URLs de route front (barre d'adresse :
 * `page.goto`, `page.url()`, `waitForURL`). Les URLs d'API (`/api/v1/players/…`)
 * ne sont PAS title-préfixées : côté backend le titre transite par le header
 * `X-LevelUp-Title` / la session, pas par le chemin — ne PAS les passer par ici.
 *
 * Voir e2e/README.md et e2e/legacy-redirect.spec.ts.
 */

/** Titre par défaut de la session serveur en E2E (bootstrap DEMO_MODE / dev). */
export const DEFAULT_TITLE_SLUG = 'halo_infinite'

/** Échappe une chaîne pour l'injecter littéralement dans une RegExp. */
function escapeRe(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** Concatène un suffixe (sans `/` de tête imposé) à une base de chemin. */
function withSuffix(base: string, suffix: string): string {
  if (!suffix) return base
  return `${base}/${suffix.replace(/^\/+/, '')}`
}

/**
 * URL de route joueur title-préfixée, titre EXPLICITE.
 *
 * @param titleSlug identifiant interne du titre (`halo_infinite`, `halo_5`, …).
 * @param playerSlug slug du joueur (`demo-player`, `JGtm`, …).
 * @param suffix segment(s) sous le joueur, SANS `/` de tête — peut porter query
 *   et hash (`home`, `stats/timeseries`, `matches/${id}`, `compare?target=X#h`).
 *   Vide = racine joueur.
 * @example titlePath('halo_5', 'JGtm', 'home') // '/t/halo_5/players/JGtm/home'
 */
export function titlePath(titleSlug: string, playerSlug: string, suffix = ''): string {
  return withSuffix(`/t/${titleSlug}/players/${playerSlug}`, suffix)
}

/**
 * URL de route joueur pour le titre par défaut ({@link DEFAULT_TITLE_SLUG}) —
 * forme utilisée par la quasi-totalité des specs.
 *
 * @example playerPath('demo-player', 'stats/timeseries')
 *   // '/t/halo_infinite/players/demo-player/stats/timeseries'
 */
export function playerPath(playerSlug: string, suffix = ''): string {
  return titlePath(DEFAULT_TITLE_SLUG, playerSlug, suffix)
}

/**
 * Ancienne URL joueur NON préfixée (`/players/{slug}/…`), telle qu'un bookmark ou
 * un lien externe l'émet encore. RÉSERVÉE aux specs qui exercent la redirection
 * legacy (legacy-redirect.spec.ts) : partout ailleurs une route front DOIT être
 * title-préfixée via {@link playerPath}.
 *
 * @example legacyPlayerPath('demo-player', 'objectifs') // '/players/demo-player/objectifs'
 */
export function legacyPlayerPath(playerSlug: string, suffix = ''): string {
  return withSuffix(`/players/${playerSlug}`, suffix)
}

/**
 * RegExp (sous-chaîne, NON ancrée) matchant une URL de route joueur du titre par
 * défaut — pour `page.waitForURL(...)` / `expect(page.url()).toMatch(...)`. Non
 * ancrée : tolère un `?search`/`#hash` en fin d'URL (une page peut estamper `?f=`
 * après l'atterrissage) sans affaiblir l'assertion sur le CHEMIN.
 *
 * @example playerUrlPattern('demo-player', 'home') // /\/t\/halo_infinite\/players\/demo-player\/home/
 */
export function playerUrlPattern(playerSlug: string, suffix = ''): RegExp {
  return new RegExp(escapeRe(playerPath(playerSlug, suffix)))
}

/**
 * RegExp matchant le SEGMENT de titre `/t/{titleSlug}/players/` quel que soit le
 * joueur — pour asserter le TITRE d'atterrissage quand le slug joueur peut être
 * réécrit par le filet `resolvePlayerFallback` du layout joueur (ex. joueur absent
 * du titre visé). Utilisé par la matrice H5 de legacy-redirect.spec.ts.
 */
export function titleSegmentPattern(titleSlug: string): RegExp {
  return new RegExp(`/t/${escapeRe(titleSlug)}/players/`)
}

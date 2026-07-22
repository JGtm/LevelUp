/**
 * Helpers de chemin relatif au joueur (D-10) — source UNIQUE autorisée du littéral
 * `/players/` pour les surfaces de navigation, verrouillée par le garde-rail ratchet.
 *
 * Contexte : le pathname title-scoped a la forme
 * `/{-lang}/t/{slug}/players/{playerSlug}{suffix}` (lang + préfixe titre présents ou
 * non selon l'URL). Le SUFFIXE (`/home`, `/stats/sessions`, `/career/citations`, …)
 * est la seule partie qui identifie la page DANS le scope joueur. Les matchers de nav
 * (section active) et la résolution de titre de page raisonnent sur ce suffixe — sans
 * recopier le littéral `/players/` (règle CLAUDE.md n°6, garde-rail
 * `no-title-literals.ratchet.test.ts`).
 */

const PLAYER_SEGMENT = /\/players\/[^/]+/

/**
 * Portion du pathname relative au joueur : ce qui suit `/players/{playerSlug}`.
 *  - `/t/halo_infinite/players/x/stats/sessions` → `/stats/sessions`
 *  - `/en/t/halo_5/players/x/home`               → `/home`
 *  - `/t/halo_infinite/players/x`                → `''` (racine joueur nue)
 *  - `/settings`, `/`                            → `null` (page agnostique)
 *
 * Tolère les anciens pathnames `/players/{slug}/…` (utile aux tests et au splat de
 * redirection legacy) : le préfixe titre/langue est optionnel dans la détection.
 */
export function playerRelativePath(pathname: string): string | null {
  const m = PLAYER_SEGMENT.exec(pathname)
  if (!m) return null
  return pathname.slice(m.index + m[0].length)
}

const TEMPLATE_MARKER = '/players/$playerSlug'

/**
 * Suffixe relatif au joueur d'un TEMPLATE de route title-scoped
 * (`/{-$lang}/t/$titleSlug/players/$playerSlug{suffix}`, typé `FileRouteTypes['to']`).
 * Retourne `''` si le template s'arrête à la racine joueur. Sert à comparer une cible
 * de route au pathname courant (via `playerRelativePath`) sans résoudre les params.
 */
export function routeTemplateSuffix(to: string): string {
  const i = to.indexOf(TEMPLATE_MARKER)
  if (i === -1) return ''
  return to.slice(i + TEMPLATE_MARKER.length)
}

/**
 * Href ABSOLU title-scoped vers une page joueur, pour les surfaces qui ne passent
 * PAS par un `<Link>`/navigate typé : liens PLEINE PAGE (`<a href>`) et deep-links
 * partageables (`?f=`). Source UNIQUE autorisée du littéral `/players/` pour ces
 * chaînes (garde-rail ratchet) — évite d'en recopier une 4e forme dans les features.
 *
 * Forme émise : `/t/{titleSlug}/players/{playerSlug}{suffix}`. Le segment langue est
 * OMIS (locale de session) : ces liens rechargent la page, donc l'ancien comportement
 * sans segment langue est préservé à l'identique. Le `suffix` doit inclure son propre
 * `?query`/`#hash` s'il y a lieu (ex. `'/stats/timeseries?f=…'`). `playerSlug` est
 * encodé (dérivé utilisateur) ; `titleSlug` est un identifiant interne sûr, verbatim.
 */
export function playerScopedHref(titleSlug: string, playerSlug: string, suffix = ''): string {
  return `/t/${titleSlug}/players/${encodeURIComponent(playerSlug)}${suffix}`
}

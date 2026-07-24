/**
 * waypointUrl — construction centralisée du lien « Ouvrir sur Halo Waypoint »
 * (I19). Title-aware : gating côté appelant via
 * `useCapability('waypoint_match_url')`, segment d'URL résolu par titre.
 *
 * Formats vérifiés (2026-07-24, fournis/valides par l'utilisateur) :
 *   - Halo Infinite : /halo-infinite/players/{gt}/matches/{id}
 *   - Halo 5        : /halo-5-guardians/players/{gt}/matches/arena/{id}
 *     (bucket "arena" constant : le corpus suivi est du matchmaking Arena —
 *     Warzone est exclu du sync, les customs ne sont pas listées).
 *
 * Centralisé à la 3e copie du pattern — cf. règle CLAUDE.md « ≤ 2 copies d'un même
 * pattern ». Appelants actuels : ExplorerMatchesTable, SquadSynergyHistoryTable.
 * Garde-rail : waypointUrl.guard.test.ts interdit toute reconstruction ad hoc du
 * domaine halowaypoint.com ailleurs dans apps/web/src.
 */
import type { UiTheme } from '@/stores/settingsDraftStore'

/** Segment de chemin Waypoint par titre. Fallback = segment Infinite (un titre
 *  futur sans page Waypoint ne déclare simplement pas la capability). */
function waypointTitleSegment(titleSlug: string): string {
  return titleSlug === 'halo_5' ? 'halo-5-guardians' : 'halo-infinite'
}

/** URL publique de la page de détail d'un match sur Halo Waypoint. */
export function buildWaypointMatchUrl(playerSlug: string, matchId: string, titleSlug = 'halo_infinite'): string {
  const seg = waypointTitleSegment(titleSlug)
  const bucket = titleSlug === 'halo_5' ? 'arena/' : ''
  return `https://www.halowaypoint.com/${seg}/players/${encodeURIComponent(playerSlug)}/matches/${bucket}${matchId}`
}

/** Chemin du logo Halo Waypoint selon le thème local — même convention que
 *  MatchNemesisCards (thème clair -> logo noir, sombre -> logo blanc). */
export function waypointLogoSrc(theme: UiTheme): string {
  return `/icons/halowaypoint-${theme === 'light' ? 'black' : 'white'}.png`
}

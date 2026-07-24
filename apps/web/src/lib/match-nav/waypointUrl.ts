/**
 * waypointUrl — construction centralisée du lien « Ouvrir sur Halo Waypoint »
 * (I19). Halo Infinite uniquement : gating côté appelant via
 * `useCapability('waypoint_match_url')` (absente pour Halo 5 — le segment
 * d'URL "halo-infinite" ne s'appliquerait pas à un autre titre).
 *
 * Centralisé au 3e point d'usage (ExplorerMatchesTable, SquadSynergyHistoryTable,
 * CareerTopMatchesTable) — cf. règle CLAUDE.md « ≤ 2 copies d'un même pattern ».
 * Garde-rail : waypointUrl.guard.test.ts interdit toute reconstruction ad hoc du
 * domaine halowaypoint.com ailleurs dans apps/web/src.
 */
import type { UiTheme } from '@/stores/settingsDraftStore'

/** URL publique de la page de détail d'un match sur Halo Waypoint. */
export function buildWaypointMatchUrl(playerSlug: string, matchId: string): string {
  return `https://www.halowaypoint.com/halo-infinite/players/${encodeURIComponent(playerSlug)}/matches/${matchId}`
}

/** Chemin du logo Halo Waypoint selon le thème local — même convention que
 *  MatchNemesisCards (thème clair -> logo noir, sombre -> logo blanc). */
export function waypointLogoSrc(theme: UiTheme): string {
  return `/icons/halowaypoint-${theme === 'light' ? 'black' : 'white'}.png`
}

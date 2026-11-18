/**
 * playerCampLabel.ts — LE NOM DU CAMP D'UN JOUEUR, tel que l'en-tête de son camp l'écrit dans la colonne des équipes.
 *
 * Le camp est celui du FILM (`ReplayPlayer.team`) ; la feuille ne fait que le nommer, par la cascade du dépôt
 * (`campLabel`) lue sur les lignes de feuille de ses membres. Un joueur sans équipe au film (`team` absent, ou
 * `-1` : un mode sans camps) n'a pas de nom de camp : `null`, rien n'est écrit à la place.
 */
import type { TeamNamingText } from '@/lib/halo/teamLabel'
import { campLabel, campSideOf } from '../../../lib/replay/replayCamps'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'

export function playerCampLabel(
  player: ReplayPlayer,
  players: readonly ReplayPlayer[],
  text: TeamNamingText,
): string | null {
  const team = player.team
  if (team === undefined || team < 0) return null
  const rows = players.filter((p) => p.team === team).map((p) => p.board)
  return campLabel({ team, side: campSideOf(rows) }, rows, text)
}

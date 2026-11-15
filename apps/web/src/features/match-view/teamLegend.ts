/**
 * teamLegend.ts — LES NOMS D'ÉQUIPE DE LA LÉGENDE des graphes divergents, et ils sont ceux de
 * « Score dans le temps » : le même résolveur de libellé (`resolveTeamLabel`), sur la même feuille.
 *
 * UN CAMP QU'ON N'A PAS MESURÉ N'A PAS DE NOM. Sans exactement une équipe du côté du joueur, ou
 * quand un camp ne se situe pas face à lui, aucun nom n'est rendu : l'appelant retombe sur son
 * texte générique (jamais un nom deviné). Face à plusieurs camps adverses, il n'y a pas de nom
 * unique : `opponent` est nul et `multiTeam` le dit.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveTeamLabel, type TeamLabelText } from '@/lib/halo/teamLabel'
import { parseTeamSideID } from '@/lib/halo/teamNames'

import { allyOfTeamId, resolveXuidMeta } from './xuidMeta'

export interface TeamLegend {
  /** Le nom de l'équipe du joueur ; nul quand elle ne se situe pas. */
  ally: string | null
  /** Le nom de l'unique équipe d'en face ; nul s'il y en a plusieurs ou aucune connue. */
  opponent: string | null
  /** Plus d'une équipe en face. */
  multiTeam: boolean
}

const UNKNOWN: TeamLegend = { ally: null, opponent: null, multiTeam: false }

/** teamLegend lit les noms d'équipe de la légende depuis la feuille du match. */
export function teamLegend(
  scoreboard: MatchScoreboardRow[] | null | undefined,
  meXUID: string | null,
  text: TeamLabelText,
): TeamLegend {
  const board = scoreboard ?? []
  const ids = [...new Set(board.map((r) => parseTeamSideID(r.team_side)).filter((id): id is number => id != null))]
  const meta = resolveXuidMeta(board, meXUID)
  const allies = ids.filter((id) => allyOfTeamId(board, meta, id) === true)
  const others = ids.filter((id) => allyOfTeamId(board, meta, id) !== true)
  if (allies.length !== 1 || others.length === 0) return UNKNOWN
  // Un camp d'en face dont l'identité ne se situe pas : rien n'est affirmé.
  if (others.some((id) => allyOfTeamId(board, meta, id) === null)) return UNKNOWN
  const labelOf = (id: number) =>
    resolveTeamLabel(
      board.filter((r) => r.team_side === `t${id}`),
      `t${id}`,
      text,
    )
  return {
    ally: labelOf(allies[0]),
    opponent: others.length === 1 ? labelOf(others[0]) : null,
    multiTeam: others.length > 1,
  }
}

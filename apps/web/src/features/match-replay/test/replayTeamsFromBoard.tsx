/**
 * replayTeamsFromBoard.tsx — LA COLONNE DES ÉQUIPES MONTÉE DEPUIS UNE FEUILLE, pour les fichiers de tests.
 *
 * `ReplayTeams` reçoit la jointure film <-> base toute faite (`model.players`, construite une fois par la
 * page). Un test, lui, a un document et une feuille : ce montage fait la jointure comme la page, avec la
 * même mémoïsation (document, feuille), et ne change rien d'autre. Il FOURNIT, il ne juge pas.
 */
import { useMemo } from 'react'
import type { ComponentProps } from 'react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { ReplayTeams } from '../ui/ReplayTeams'

type TeamsProps = Omit<ComponentProps<typeof ReplayTeams>, 'players'> & {
  /** La feuille du match ; `[]` = film seul. */
  scoreboard: MatchScoreboardRow[]
}

export function ReplayTeamsFromBoard({ scoreboard, ...rest }: TeamsProps) {
  const players = useMemo(() => buildPlayers(rest.doc, scoreboard), [rest.doc, scoreboard])
  return <ReplayTeams {...rest} players={players} />
}

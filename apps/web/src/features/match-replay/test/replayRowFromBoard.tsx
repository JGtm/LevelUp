/**
 * replayRowFromBoard.tsx — LA RANGÉE DES FICHES MONTÉE DEPUIS UNE FEUILLE, pour les fichiers de tests.
 *
 * `ReplayTeamsRow` reçoit la jointure film <-> base toute faite (`model.players`, construite une fois par la
 * page). Un test, lui, a un document et une feuille : ce montage fait la jointure comme la page, avec la
 * même mémoïsation (document, feuille), et ne change rien d'autre. Il monte la forme « fiches » (une
 * `ReplayPlayerCard` au gabarit compact par place), sans bandeau de score ; le joueur suivi se désigne par
 * `watched` (clé de feuille ou de roster). Il FOURNIT, il ne juge pas. La rangée n'admet que deux camps :
 * un document à un seul camp ne rend rien.
 */
import { useMemo } from 'react'
import type { ComponentProps } from 'react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { ReplayTeamsRow } from '../ui/ReplayTeamsRow'
import { trackedPlayerOf } from '../model/playerPanelLogic'

type RowProps = Omit<ComponentProps<typeof ReplayTeamsRow>, 'players' | 'tracked' | 'score' | 'grandeEquipe'> & {
  /** La feuille du match ; `[]` = film seul. */
  scoreboard: MatchScoreboardRow[]
  /** Le joueur regardé par la page (clé de feuille ou de roster) ; `null` = personne. */
  watched?: string | null
}

export function ReplayRowFromBoard({ scoreboard, watched = null, ...rest }: RowProps) {
  const players = useMemo(() => buildPlayers(rest.doc, scoreboard), [rest.doc, scoreboard])
  const tracked = useMemo(() => trackedPlayerOf(watched, players), [watched, players])
  return <ReplayTeamsRow {...rest} players={players} tracked={tracked} score={null} grandeEquipe={false} />
}

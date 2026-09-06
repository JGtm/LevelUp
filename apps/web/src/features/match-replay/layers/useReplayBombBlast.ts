/**
 * useReplayBombBlast — LE CÂBLAGE de la DÉFLAGRATION D'ASSAUT (`bomb_detonations`), en un point.
 *
 * MÊME PARTI QUE `useReplayVipCrown` et `useReplaySkullCarrier` : le canvas du rejeu porte une
 * dette de taille GELÉE par un seuil (`max-lines` eslint, R5) — toute addition s'y
 * fait par EXTRACTION, et ce hook n'en rend au canvas que deux lignes utiles.
 *
 * POURQUOI PAS DANS `useReplayFlagCarries`, qui porte déjà l'onde de capture et la même
 * jointure. Parce que ce hook-là est gardé par `carries.length === 0` : un match d'Assaut ne
 * publie AUCUN drapeau, donc l'explosion n'y serait jamais peinte. Deux modes disjoints, deux
 * gardes disjointes — les fondre ferait dépendre l'explosion d'une donnée qui n'existe pas dans
 * son mode.
 *
 * AUCUNE GARDE DE MODE ICI, et c'est délibéré : la garde EST la donnée. `bomb_detonations` n'est
 * publié que par un match d'Assaut (`ObjectiveTypeBomb`, cf. `objectiveevents/named.go`), donc
 * un film d'un autre mode rend une liste vide sans qu'on ait à connaître sa variante. Même
 * doctrine que le crâne libre et la couronne : le calque lit ce que le document porte.
 */
import { useCallback, useMemo } from 'react'

import type { MatchScoreboardRow } from '@/lib/api/types'

import {
  BOMB_BLAST_HOLD_FRAMES,
  buildBombBlastFx,
  drawBombBlastFx,
  type BombBlastStyle,
} from './bombBlastFx'
import { useCarrierPosAt } from '../model/carrierPosition'
import { type CanvasView } from '../model/replayView'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { allyTeamFromScoreboard, teamOfXuidFromScoreboard } from '../model/matchSides'

export interface BombBlastHookInput {
  doc: ReplayDocumentReady
  view: CanvasView
  scoreboard: MatchScoreboardRow[] | null | undefined
  /** Encre d'un camp vu de la page (tokens déjà résolus par l'appelant). */
  teamColorOf: (ally: boolean) => string
  /** Encre servie quand le camp est inconnu : ni équipe inventée, ni explosion invisible. */
  neutral: string
  reducedMotion: boolean
}

export interface ReplayBombBlast {
  /** Le film porte-t-il des explosions ? Sert au canvas à ne rien peindre pour rien. */
  available: boolean
  /** Peint les déflagrations de l'image demandée. No-op quand il n'y en a aucune. */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
}

export function useReplayBombBlast({
  doc,
  view,
  scoreboard,
  teamColorOf,
  neutral,
  reducedMotion,
}: BombBlastHookInput): ReplayBombBlast {
  // La relecture de position partagée (carrierPosition.ts : embarqué -> position du véhicule,
  // sinon celle du bipède) — la fenêtre après-mort du repli compte PLUS ici qu'ailleurs : le
  // poseur meurt souvent DANS son explosion.
  const posOf = useCarrierPosAt(doc)

  const blasts = useMemo(() => buildBombBlastFx(doc, posOf), [doc, posOf])

  // LE CAMP DE L'AUTEUR SE LIT AU TABLEAU DE BORD, jamais dans le film : l'action ne porte que
  // le xuid. Un auteur absent du tableau prend le neutre du thème — jamais une équipe devinée,
  // même règle que l'onde de capture.
  const teamOfXuid = useMemo(() => teamOfXuidFromScoreboard(scoreboard), [scoreboard])

  const allyTeamID = useMemo(() => allyTeamFromScoreboard(scoreboard), [scoreboard])

  const style = useMemo<BombBlastStyle>(
    () => ({
      inkOf: (xuid: string) => {
        const team = teamOfXuid.get(xuid)
        if (team === undefined || allyTeamID === null) return neutral
        return teamColorOf(team === allyTeamID)
      },
      reducedMotion,
    }),
    [teamOfXuid, allyTeamID, teamColorOf, neutral, reducedMotion],
  )

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      if (blasts.length === 0) return
      drawBombBlastFx(ctx, blasts, view, { frame, hold: BOMB_BLAST_HOLD_FRAMES }, style)
    },
    [blasts, view, style],
  )

  return { available: blasts.length > 0, paint }
}

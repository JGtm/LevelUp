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

import {
  BOMB_BLAST_HOLD_FRAMES,
  buildBombBlastFx,
  drawBombBlastFx,
  type BombBlastStyle,
} from './bombBlastFx'
import { useCarrierPosAt } from '../model/carrierPosition'
import { type CanvasView } from '../model/replayView'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { FilmAllegiance } from '../../../lib/replay/filmAllegiance'

export interface BombBlastHookInput {
  doc: ReplayDocumentReady
  view: CanvasView
  /**
   * L'allégeance lue dans le film, vue du point de vue (`model.allegiance`) : le camp de
   * l'auteur d'une déflagration. REQUISE : elle porte le point de vue (revue F4 du 2026-09-07).
   */
  allegiance: FilmAllegiance
  /** Encre d'un camp vu de la page (tokens déjà résolus par l'appelant). */
  teamColorOf: (ally: boolean) => string
  /** Encre servie quand le camp est inconnu : ni équipe inventée, ni explosion invisible. */
  neutral: string
  reducedMotion: boolean
}

export interface ReplayBombBlast {
  /** Le film porte-t-il des explosions ? Sert au canvas à ne rien peindre pour rien. */
  available: boolean
  /**
   * LE NOM DU CALQUE, porté par le calque et non par la table de liaison du canvas
   * (2026-09-06, revue R1 constat C2) : c'est ce qui rend impossible de peindre ce geste
   * sous l'identité d'un autre calque.
   */
  id: 'deflagration'
  /** Peint les déflagrations de l'image demandée. No-op quand il n'y en a aucune. */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
}

export function useReplayBombBlast({
  doc,
  view,
  allegiance,
  teamColorOf,
  neutral,
  reducedMotion,
}: BombBlastHookInput): ReplayBombBlast {
  // La relecture de position partagée (carrierPosition.ts : embarqué -> position du véhicule,
  // sinon celle du bipède) — la fenêtre après-mort du repli compte PLUS ici qu'ailleurs : le
  // poseur meurt souvent DANS son explosion.
  const posOf = useCarrierPosAt(doc)

  const blasts = useMemo(() => buildBombBlastFx(doc, posOf), [doc, posOf])

  // LE CAMP DE L'AUTEUR EST SON ALLÉGEANCE DU FILM : l'action ne porte que son xuid, un bot s'y
  // relie par sa ligne de feuille. Sans allégeance (le film tait son équipe, ou celle du joueur
  // regardé), le neutre du thème — jamais une équipe devinée, même règle que l'onde de capture.
  const style = useMemo<BombBlastStyle>(
    () => ({
      inkOf: (xuid: string) => {
        const ally = allegiance.ofXuid(xuid)
        return ally === null ? neutral : teamColorOf(ally)
      },
      reducedMotion,
    }),
    [allegiance, teamColorOf, neutral, reducedMotion],
  )

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      if (blasts.length === 0) return
      drawBombBlastFx(ctx, blasts, view, { frame, hold: BOMB_BLAST_HOLD_FRAMES }, style)
    },
    [blasts, view, style],
  )

  return { id: 'deflagration', available: blasts.length > 0, paint }
}

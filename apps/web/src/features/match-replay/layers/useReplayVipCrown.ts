/**
 * useReplayVipCrown — LE CÂBLAGE du calque de la COURONNE VIP (schéma 22), en un seul point.
 *
 * MÊME PARTI QUE `useReplayFlagCarries` et `useReplayObjectiveObjects` : le canvas du rejeu porte
 * une dette de taille GELÉE par un seuil (`max-lines` eslint, R5) — toute addition s'y
 * fait par EXTRACTION. Ce hook réunit la relecture de position du VIP et le tracé par image, et
 * n'en rend au canvas que deux lignes utiles.
 *
 * LE VIP SE RELIT DANS SES TRAJECTOIRES, image par image (`useCarrierPosAt`, le même utilitaire
 * que la bombe, le crâne et le drapeau porté) : la couronne « colle » ainsi à son marqueur, alors
 * que la période ne publie qu'un intervalle. Sans position, la couronne ne se dessine pas — elle
 * n'a pas de place propre. VIP EMBARQUÉ : c'est la position du VÉHICULE qui répond (décision
 * produit du 2026-09-05, cf. l'en-tête de `carrierPosition.ts`).
 *
 * L'ENCRE SUIT LE CAMP DU VIP VU DU JOUEUR REGARDÉ (`FilmAllegiance.ofXuid`), comme le drapeau :
 * la couleur qu'il a choisie pour allié et adverse. Sans allégeance, le neutre du thème.
 */
import { useCallback, useMemo } from 'react'

import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'

import { useCarrierPosAt } from '../model/carrierPosition'
import { type CanvasView } from '../model/replayView'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { drawVipCrown, type VipCrownInput } from './vipCrownLayer'

interface UseReplayVipCrownArgs {
  doc: ReplayDocumentReady
  view: CanvasView
  /** Faux quand le calque est éteint : rien n'est dessiné. */
  enabled: boolean
  /** L'allégeance lue dans le film, vue du point de vue : le camp de chaque VIP. */
  allegiance: Pick<FilmAllegiance, 'ofXuid'>
  /** Encre d'un camp vu de la page (tokens déjà résolus par l'appelant). */
  teamColorOf: (ally: boolean) => string
  /** Encre servie quand le camp est inconnu (token du thème). */
  neutral: string
  /** Encre du FOND, pour le liseré (la même que le drapeau et le crâne). */
  outline: string
  reducedMotion: boolean
}

export interface ReplayVipCrown {
  /** Le film porte-t-il des périodes VIP ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /**
   * LE NOM DU CALQUE, porté par le calque et non par la table de liaison du canvas
   * (2026-09-06, revue R1 constat C2) : c'est ce qui rend impossible de peindre ce geste
   * sous l'identité d'un autre calque.
   */
  id: 'couronne-vip'
  /** Peint la couronne à l'image demandée. No-op quand il n'y en a aucune, ou qu'elle est éteinte. */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
}

export function useReplayVipCrown({
  doc,
  view,
  enabled,
  allegiance,
  teamColorOf,
  neutral,
  outline,
  reducedMotion,
}: UseReplayVipCrownArgs): ReplayVipCrown {
  const periods = doc.vipCrown

  // La relecture de position partagée (carrierPosition.ts) : embarqué -> position du véhicule,
  // sinon celle du bipède.
  const posOf = useCarrierPosAt(doc)

  const inkOf = useCallback(
    (xuid: string) => {
      const ally = allegiance.ofXuid(xuid)
      return ally === null ? neutral : teamColorOf(ally)
    },
    [allegiance, neutral, teamColorOf],
  )

  const layer = useMemo<VipCrownInput>(
    () => ({ style: { inkOf, outline, reducedMotion }, posOf }),
    [inkOf, outline, reducedMotion, posOf],
  )

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      if (!enabled || periods.length === 0) return
      drawVipCrown(ctx, layer, periods, view, frame)
    },
    [enabled, periods, layer, view],
  )

  return { id: 'couronne-vip', available: periods.length > 0, paint }
}

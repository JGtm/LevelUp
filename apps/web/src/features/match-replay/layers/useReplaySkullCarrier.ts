/**
 * useReplaySkullCarrier — LE CÂBLAGE du calque du PORTEUR DU CRÂNE d'Oddball (schéma 23), en un
 * seul point.
 *
 * MÊME PARTI QUE `useReplayVipCrown` et `useReplayFlagCarries` : le canvas du rejeu porte une dette
 * de taille GELÉE par un seuil (`max-lines` eslint, R5) — toute addition s'y fait par
 * EXTRACTION. Ce hook réunit la relecture de position du porteur et le tracé par image, et n'en
 * rend au canvas que deux lignes utiles.
 *
 * LE PORTEUR SE RELIT DANS SES TRAJECTOIRES, image par image (`useCarrierPosAt`, le même
 * utilitaire que la bombe, le drapeau porté et la couronne) : le crâne « colle » ainsi à son
 * marqueur, alors que la période ne publie qu'un intervalle. Sans position, le crâne ne se dessine
 * pas — il n'a pas de place propre (il est TOUJOURS sur le joueur qui le porte). PORTEUR
 * EMBARQUÉ : c'est la position du VÉHICULE qui répond (décision produit du 2026-09-05, cf.
 * l'en-tête de `carrierPosition.ts`).
 */
import { useCallback, useMemo } from 'react'

import { useCarrierPosAt } from '../model/carrierPosition'
import { type CanvasView } from '../model/replayView'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { drawSkullCarrier, type SkullCarrierInput } from './skullCarrierLayer'

interface UseReplaySkullCarrierArgs {
  doc: ReplayDocumentReady
  view: CanvasView
  /** Faux quand le calque est éteint : rien n'est dessiné. */
  enabled: boolean
  /** L'encre du crâne, déjà résolue par l'appelant (token du thème). */
  ink: string
  /** L'encre du FOND : le liseré du crâne, comme celui du drapeau (déjà résolue par l'appelant). */
  outline: string
  reducedMotion: boolean
}

export interface ReplaySkullCarrier {
  /** Le film porte-t-il des portages ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /**
   * LE NOM DU CALQUE, porté par le calque et non par la table de liaison du canvas
   * (2026-09-06, revue R1 constat C2) : c'est ce qui rend impossible de peindre ce geste
   * sous l'identité d'un autre calque.
   */
  id: 'crane-porte'
  /** Peint le crâne à l'image demandée. No-op quand il n'y en a aucun, ou qu'il est éteint. */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
}

export function useReplaySkullCarrier({
  doc,
  view,
  enabled,
  ink,
  outline,
  reducedMotion,
}: UseReplaySkullCarrierArgs): ReplaySkullCarrier {
  const carries = doc.skullCarries

  // La relecture de position partagée (carrierPosition.ts) : embarqué -> position du véhicule,
  // sinon celle du bipède.
  const posOf = useCarrierPosAt(doc)

  const layer = useMemo<SkullCarrierInput>(
    () => ({ style: { ink, outline, reducedMotion }, posOf }),
    [ink, outline, reducedMotion, posOf],
  )

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      if (!enabled || carries.length === 0) return
      drawSkullCarrier(ctx, layer, carries, view, frame)
    },
    [enabled, carries, layer, view],
  )

  return { id: 'crane-porte', available: carries.length > 0, paint }
}

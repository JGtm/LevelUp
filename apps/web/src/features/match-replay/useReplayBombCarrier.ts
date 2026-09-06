/**
 * useReplayBombCarrier — LE CÂBLAGE du calque de LA BOMBE D'ASSAUT (schéma 30), en un seul
 * point : portée sur son porteur, posée au sol entre un lâcher et la prise suivante.
 *
 * MÊME PARTI QUE `useReplaySkullCarrier` : le canvas du rejeu porte une dette de taille GELÉE
 * par un cliquet (`placementFamily.guard.test.ts`) — toute addition s'y fait par EXTRACTION.
 * Ce hook réunit la relecture de position, la garde d'horloge des explosions et le tracé par
 * image, et n'en rend au canvas que deux lignes utiles.
 *
 * LE PORTEUR SE RELIT DANS SES TRAJECTOIRES, image par image (`useCarrierPosAt`, le même
 * utilitaire que le crâne, le drapeau porté et la couronne) : la bombe « colle » à son
 * marqueur. La bombe AU SOL relit la position du LÂCHEUR à l'instant du lâcher (t1) — la
 * fenêtre après-mort compte ici : un porteur tué lâche la bombe à sa mort, et sa position doit
 * se lire à cette image-là. PORTEUR EMBARQUÉ : c'est la position du VÉHICULE qui répond
 * (décision produit du 2026-09-05, cf. l'en-tête de `carrierPosition.ts`) — un bipède attaché
 * ne réplique plus, et la bombe traversait le décor en ligne droite.
 *
 * LES EXPLOSIONS COUPENT LE SOL, sous la garde d'horloge de la déflagration
 * (`filmClockTrusted`, même règle que `bombBlastFx`) : sans origine résolue, `explosions`
 * vaut `null` et le calque ne dessine PAS de bombe au sol — jamais une bombe qui survivrait
 * à sa propre explosion.
 */
import { useCallback, useMemo } from 'react'

import { filmClockTrusted } from '@/lib/replay/scoreTimeline'

import { BOMB_DETONATION_STAT } from './bombBlastFx'
import { drawBombCarrier, type BombCarrierInput } from './bombCarrierLayer'
import { useCarrierPosAt } from './carrierPosition'
import { type CanvasView } from './replayView'
import type { ReplayDocumentReady } from './replayNormalize'

interface UseReplayBombCarrierArgs {
  doc: ReplayDocumentReady
  view: CanvasView
  /** Faux quand le calque est éteint : rien n'est dessiné. */
  enabled: boolean
  /** L'encre de la bombe, déjà résolue par l'appelant (token du thème). */
  ink: string
  /** L'encre du FOND : le liseré, comme celui du crâne (déjà résolue par l'appelant). */
  outline: string
  reducedMotion: boolean
}

export interface ReplayBombCarrier {
  /** Le film porte-t-il des portages ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /** Peint la bombe à l'image demandée. No-op quand il n'y en a aucune, ou qu'elle est éteinte. */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
}

export function useReplayBombCarrier({
  doc,
  view,
  enabled,
  ink,
  outline,
  reducedMotion,
}: UseReplayBombCarrierArgs): ReplayBombCarrier {
  const carries = doc.bombCarries

  // La relecture de position partagée (carrierPosition.ts : embarqué -> position du véhicule,
  // sinon celle du bipède) : un porteur tué lâche à sa mort, la fenêtre après-mort du repli
  // garde sa dernière position lisible.
  const posOf = useCarrierPosAt(doc)

  // LES FRAMES DES EXPLOSIONS, triées — `null` sans horloge de confiance : le sol s'éteint
  // (cf. bombCarrierLayer, en-tête). Le porté, lui, ne dépend pas de cette horloge : les
  // périodes sont déjà des frames du document.
  const explosions = useMemo<readonly number[] | null>(() => {
    if (!filmClockTrusted(doc)) return null
    const out: number[] = []
    for (const a of doc.objectives) {
      if (a.stat === BOMB_DETONATION_STAT) out.push(a.t)
    }
    return out.sort((a, b) => a - b)
  }, [doc])

  const layer = useMemo<BombCarrierInput>(
    () => ({ style: { ink, outline, reducedMotion }, posOf }),
    [ink, outline, reducedMotion, posOf],
  )

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      if (!enabled || carries.length === 0) return
      drawBombCarrier(ctx, layer, carries, explosions, view, frame)
    },
    [enabled, carries, explosions, layer, view],
  )

  return { available: carries.length > 0, paint }
}

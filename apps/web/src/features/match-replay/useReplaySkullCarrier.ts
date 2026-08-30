/**
 * useReplaySkullCarrier — LE CÂBLAGE du calque du PORTEUR DU CRÂNE d'Oddball (schéma 23), en un
 * seul point.
 *
 * MÊME PARTI QUE `useReplayVipCrown` et `useReplayFlagCarries` : le canvas du rejeu porte une dette
 * de taille GELÉE par un cliquet (`placementFamily.guard.test.ts`) — toute addition s'y fait par
 * EXTRACTION. Ce hook réunit la relecture de position du porteur et le tracé par image, et n'en
 * rend au canvas que deux lignes utiles.
 *
 * LE PORTEUR SE RELIT DANS SES TRAJECTOIRES, image par image (`posOfPlayerAt`, le même utilitaire
 * que les effets de mort, le drapeau porté et la couronne) : le crâne « colle » ainsi à son
 * marqueur, alors que la période ne publie qu'un intervalle. Sans position, le crâne ne se dessine
 * pas — il n'a pas de place propre (il est TOUJOURS sur le joueur qui le porte).
 */
import { useCallback, useMemo } from 'react'

import { KILLPOS_WINDOW_MS, posOfPlayerAt } from './killFx'
import type { CanvasView } from './objectivesLayer'
import { msToFrames } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'
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

  // LES VIES PAR JOUEUR, indexées une fois : la relecture de position tourne à chaque image — la
  // refaire dans la boucle balaierait toutes les traces du film.
  const livesByXuid = useMemo(() => {
    const map = new Map<string, ReplayTrackReady[]>()
    for (const t of doc.tracks) {
      if (!t.xuid) continue
      const list = map.get(t.xuid)
      if (list) list.push(t)
      else map.set(t.xuid, [t])
    }
    return map
  }, [doc.tracks])

  const deathFrames = useMemo(
    () => Math.max(1, Math.round(msToFrames(KILLPOS_WINDOW_MS, doc))),
    [doc],
  )
  const posOf = useCallback(
    (xuid: string, frame: number) => posOfPlayerAt(livesByXuid.get(xuid), frame, deathFrames),
    [livesByXuid, deathFrames],
  )

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

  return { available: carries.length > 0, paint }
}

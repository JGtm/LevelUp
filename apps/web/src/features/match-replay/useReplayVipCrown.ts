/**
 * useReplayVipCrown — LE CÂBLAGE du calque de la COURONNE VIP (schéma 22), en un seul point.
 *
 * MÊME PARTI QUE `useReplayFlagCarries` et `useReplayObjectiveObjects` : le canvas du rejeu porte
 * une dette de taille GELÉE par un cliquet (`placementFamily.guard.test.ts`) — toute addition s'y
 * fait par EXTRACTION. Ce hook réunit la relecture de position du VIP et le tracé par image, et
 * n'en rend au canvas que deux lignes utiles.
 *
 * LE VIP SE RELIT DANS SES TRAJECTOIRES, image par image (`posOfPlayerAt`, le même utilitaire que
 * les effets de mort et le drapeau porté) : la couronne « colle » ainsi à son marqueur, alors que
 * la période ne publie qu'un intervalle. Sans position, la couronne ne se dessine pas — elle n'a
 * pas de place propre.
 */
import { useCallback, useMemo } from 'react'

import { KILLPOS_WINDOW_MS, posOfPlayerAt } from './killFx'
import type { CanvasView } from './objectivesLayer'
import { msToFrames } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'
import { drawVipCrown, type VipCrownInput } from './vipCrownLayer'

interface UseReplayVipCrownArgs {
  doc: ReplayDocumentReady
  view: CanvasView
  /** Faux quand le calque est éteint : rien n'est dessiné. */
  enabled: boolean
  /** L'encre de la couronne, déjà résolue par l'appelant (token du thème). */
  ink: string
  reducedMotion: boolean
}

export interface ReplayVipCrown {
  /** Le film porte-t-il des périodes VIP ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /** Peint la couronne à l'image demandée. No-op quand il n'y en a aucune, ou qu'elle est éteinte. */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
}

export function useReplayVipCrown({
  doc,
  view,
  enabled,
  ink,
  reducedMotion,
}: UseReplayVipCrownArgs): ReplayVipCrown {
  const periods = doc.vipCrown

  // LES VIES PAR JOUEUR, indexées une fois : la relecture de position tourne à chaque image et
  // pour chaque VIP — la refaire dans la boucle balaierait toutes les traces du film.
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

  const layer = useMemo<VipCrownInput>(
    () => ({ style: { ink, reducedMotion }, posOf }),
    [ink, reducedMotion, posOf],
  )

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      if (!enabled || periods.length === 0) return
      drawVipCrown(ctx, layer, periods, view, frame)
    },
    [enabled, periods, layer, view],
  )

  return { available: periods.length > 0, paint }
}

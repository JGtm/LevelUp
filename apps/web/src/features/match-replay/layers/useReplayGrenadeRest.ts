/**
 * useReplayGrenadeRest — LE CÂBLAGE de la FIN DE VOL des grenades, en un seul point.
 *
 * DIX-SEPTIÈME EXTRACTION du canvas du rejeu, et elle PAIE une addition : la déflagration
 * d'Assaut (`useReplayBombBlast`, 2026-08-31) demandait ses lignes de glue alors que
 * `ReplayCanvas.tsx` était PILE à son plafond (678). La doctrine du cliquet est explicite —
 * « toute addition s'y fait par extraction, le plafond suit le fichier vers le bas ».
 *
 * CE QUI SE DÉPLACE : l'appel de tracé et son objet d'options, rien d'autre. La MESURE reste où
 * elle est (`useReplayFx` cuit `grenadeRestFx`, `useReplayTiming` rend `restWindow`) — ce hook
 * ne fait que rassembler ce que le canvas recopiait, et pas une ligne de logique ne bouge.
 *
 * POURQUOI `dpr` EST UN ARGUMENT DE `paint` ET PAS DU HOOK : il est calculé DANS la boucle de
 * tracé, à chaque image, depuis `devicePixelRatio` et l'échelle d'export courante. Le figer à
 * la construction du hook servirait une épaisseur de trait périmée dès le premier export ou le
 * premier changement d'écran.
 */
import { useCallback } from 'react'

import type { FxInk } from './fxInk'
import type { GrenadeRestFx } from '../model/grenadeFx'
import { drawGrenadeRestLayer } from './grenadeRestLayer'
import { type CanvasView } from '../model/replayView'
import { frameToMs } from '../model/replayLogic'
import type { ReplayDocumentReady } from '../model/replayNormalize'

export interface GrenadeRestHookInput {
  doc: ReplayDocumentReady
  view: CanvasView
  /** Les fins de vol cuites par `useReplayFx`. */
  fx: GrenadeRestFx[]
  /** Les deux tenues, rendues par `useReplayTiming`. */
  window: { holdHalo: number; holdDynamo: number }
  /** Encres déjà résolues par l'appelant (règle color-tokens). */
  ink: FxInk
  smoke: string
  halo: string
  reducedMotion: boolean
}

export interface ReplayGrenadeRest {
  /** Peint les fins de vol de l'image demandée. No-op quand il n'y en a aucune. */
  paint: (ctx: CanvasRenderingContext2D, frame: number, dpr: number) => void
}

export function useReplayGrenadeRest({
  doc,
  view,
  fx,
  window,
  ink,
  smoke,
  halo,
  reducedMotion,
}: GrenadeRestHookInput): ReplayGrenadeRest {
  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number, dpr: number) => {
      if (fx.length === 0) return
      drawGrenadeRestLayer(
        ctx,
        fx,
        view,
        {
          frame,
          holdHalo: window.holdHalo,
          holdDynamo: window.holdDynamo,
          // Durée réelle d'UNE frame : `frameToMs` porte déjà le repli des artefacts sans
          // échelle temporelle. La lire ici plutôt que le champ brut évite qu'une explosion
          // reste figée à l'âge zéro sur un artefact ancien.
          frameMs: frameToMs(1, doc),
        },
        { ink, smoke, halo, k: dpr, reducedMotion },
      )
    },
    [fx, view, window.holdHalo, window.holdDynamo, doc, ink, smoke, halo, reducedMotion],
  )
  return { paint }
}

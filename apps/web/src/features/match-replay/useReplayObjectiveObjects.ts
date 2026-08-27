/**
 * useReplayObjectiveObjects — LE CÂBLAGE du calque des objets d'objectif LIBRES (schéma 21).
 *
 * DIXIÈME EXTRACTION IMPOSÉE PAR LE CLIQUET DE `ReplayCanvas.tsx` (cf. placementFamily.guard).
 * Le canvas garde le DESSIN ; ce hook porte l'ENCRE et la décision de peindre, exactement comme
 * `useReplayFlagCarries` porte celles des drapeaux.
 *
 * L'ENCRE EST NEUTRE, ET C'EST UNE CONSÉQUENCE DE LA MESURE, pas un choix graphique. Le document
 * ne publie AUCUN porteur pour le crâne : l'oracle a été mesuré puis réfuté (phase D4 — 40,6 à
 * 66,7 % de trous à porteur unique pour un seuil de 90 %, témoin hors trou à 66,7 et 71,4 %).
 * Une encre d'équipe afficherait donc une appartenance que la mesure refuse ; le neutre du thème
 * dit la seule chose que l'artefact garantisse — l'objet est là, il n'est à personne.
 */
import { useCallback, useMemo } from 'react'

import { drawObjectiveObjects, type ObjectiveObjectsInput } from './objectiveObjectsLayer'

import type { CanvasView } from './objectivesLayer'
import type { ReplayObjectiveObjectReady } from './replayNormalize'

interface UseReplayObjectiveObjectsArgs {
  /** Les vies libres publiées par l'artefact. Vide = rien à peindre, et c'est le cas nominal. */
  lives: readonly ReplayObjectiveObjectReady[]
  view: CanvasView
  /** L'encre neutre du thème (remplissage) et celle du liseré, déjà résolues par l'appelant. */
  ink: string
  edge: string
}

export interface ReplayObjectiveObjects {
  /** Peint les objets qui répliquent à cette image. No-op quand il n'y en a aucun. */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
}

export function useReplayObjectiveObjects({
  lives, view, ink, edge,
}: UseReplayObjectiveObjectsArgs): ReplayObjectiveObjects {
  const layer = useMemo<ObjectiveObjectsInput>(() => ({ style: { ink, edge } }), [ink, edge])
  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      if (lives.length === 0) return
      drawObjectiveObjects(ctx, layer, lives, view, frame)
    },
    [lives, layer, view],
  )
  return { paint }
}

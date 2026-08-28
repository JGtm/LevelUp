/**
 * useReplayObjectiveObjects — LE CÂBLAGE du crâne d'Oddball au sol (schémas 21 + 23).
 *
 * DIXIÈME EXTRACTION IMPOSÉE PAR LE CLIQUET DE `ReplayCanvas.tsx` (cf. placementFamily.guard).
 * Le canvas garde le DESSIN ; ce hook porte l'ENCRE et la décision de peindre, exactement comme
 * `useReplayFlagCarries` porte celles des drapeaux.
 *
 * IL NE DÉCIDE PLUS SEUL. Depuis le schéma 23, la présence du crâne se résout par
 * `skullPresenceAt(lives, carries, frame)` : le hook peint UNIQUEMENT une présence `free`, à la
 * position qu'elle porte. Le portage (`carried`) est dessiné par `skullCarrierLayer` ; l'absence
 * (respawn, retombée, pré-émission) ne dessine rien. Ce hook TIENT donc désormais le crâne à son
 * dernier repos quand une PRISE le corrobore — le trou de repos-socle n'est plus un clignotement.
 * Avec `carries: []` (artefacts pré-schéma-23), la présence retombe sur la vie active seule : le
 * rendu est alors identique au comportement historique.
 *
 * L'ENCRE EST NEUTRE, ET C'EST UNE CONSÉQUENCE DE LA MESURE, pas un choix graphique. Le document
 * ne publie AUCUN porteur pour le crâne LIBRE : l'oracle a été mesuré puis réfuté (phase D4 — 40,6
 * à 66,7 % de trous à porteur unique pour un seuil de 90 %, témoin hors trou à 66,7 et 71,4 %).
 * Une encre d'équipe afficherait donc une appartenance que la mesure refuse ; le neutre du thème
 * dit la seule chose que l'artefact garantisse — l'objet est là, il n'est à personne.
 */
import { useCallback, useMemo } from 'react'

import { drawFreeSkull, type ObjectiveObjectsInput } from './objectiveObjectsLayer'
import { skullPresenceAt } from './skullPresence'

import type { CanvasView } from './objectivesLayer'
import type { ReplayObjectiveObjectReady, ReplaySkullCarry } from './replayNormalize'

interface UseReplayObjectiveObjectsArgs {
  /** Les vies libres publiées par l'artefact. Vide = rien à peindre, et c'est le cas nominal. */
  lives: readonly ReplayObjectiveObjectReady[]
  /** Les périodes de portage (schéma 23). Vide = artefact ancien : la présence retombe sur les vies. */
  carries: readonly ReplaySkullCarry[]
  view: CanvasView
  /** L'encre neutre du thème (remplissage) et celle du FOND (liseré), déjà résolues par l'appelant. */
  ink: string
  outline: string
}

export interface ReplayObjectiveObjects {
  /** Peint le crâne libre si sa présence est `free` à cette image. No-op sinon (porté / absent). */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
}

export function useReplayObjectiveObjects({
  lives, carries, view, ink, outline,
}: UseReplayObjectiveObjectsArgs): ReplayObjectiveObjects {
  const layer = useMemo<ObjectiveObjectsInput>(() => ({ style: { ink, outline } }), [ink, outline])
  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      // Sans aucune vie émise, la présence ne peut jamais être `free` (ni active, ni repos tenu).
      if (lives.length === 0) return
      const presence = skullPresenceAt(lives, carries, frame)
      if (presence.state !== 'free') return
      drawFreeSkull(ctx, layer, presence.at, view, presence.rolling)
    },
    [lives, carries, layer, view],
  )
  return { paint }
}

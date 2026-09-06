/**
 * useReplayObjectiveObjects — LE CÂBLAGE du crâne d'Oddball au sol (schémas 21 + 23).
 *
 * DIXIÈME EXTRACTION IMPOSÉE PAR LE SEUIL DE `ReplayCanvas.tsx` (cf. `max-lines` eslint, R5).
 * Le canvas garde le DESSIN ; ce hook porte l'ENCRE et la décision de peindre, exactement comme
 * `useReplayFlagCarries` porte celles des drapeaux.
 *
 * IL NE DÉCIDE PLUS SEUL. Depuis le schéma 23, la présence du crâne se résout par
 * `skullPresenceAt(lives, carries, frame, socle)` : le hook peint UNIQUEMENT une présence `free`,
 * à la position qu'elle porte. Le portage (`carried`) est dessiné par `skullCarrierLayer`. Le crâne
 * au REPOS est posé sur son SOCLE (`skullSocle`, lu une fois) — avant sa première prise et pendant
 * les cooldowns de respawn hors-zone, il est chez lui, plus un trou invisible. Il ne reste `absent`
 * (rien dessiné) que sans socle identifiable. Avec `carries: []` (artefacts pré-schéma-23), la
 * présence retombe sur la vie active seule.
 *
 * L'ENCRE EST NEUTRE, ET C'EST UNE CONSÉQUENCE DE LA MESURE, pas un choix graphique. Le document
 * ne publie AUCUN porteur pour le crâne LIBRE : l'oracle a été mesuré puis réfuté (phase D4 — 40,6
 * à 66,7 % de trous à porteur unique pour un seuil de 90 %, témoin hors trou à 66,7 et 71,4 %).
 * Une encre d'équipe afficherait donc une appartenance que la mesure refuse ; le neutre du thème
 * dit la seule chose que l'artefact garantisse — l'objet est là, il n'est à personne.
 */
import { useCallback, useMemo } from 'react'

import { drawFreeSkull, type ObjectiveObjectsInput } from './objectiveObjectsLayer'
import { skullPresenceAt, skullSocle } from './skullPresence'

import { type CanvasView } from './replayView'
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
  // Le socle (point de réapparition) se lit UNE fois : le crâne au repos y est posé (cf. skullPresence).
  const socle = useMemo(() => skullSocle(lives), [lives])
  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      // Sans aucune vie émise, on ne connaît ni position ni socle : rien à peindre.
      if (lives.length === 0) return
      const presence = skullPresenceAt(lives, carries, frame, socle)
      if (presence.state !== 'free') return
      drawFreeSkull(ctx, layer, presence.at, view, presence.rolling)
    },
    [lives, carries, socle, layer, view],
  )
  return { paint }
}

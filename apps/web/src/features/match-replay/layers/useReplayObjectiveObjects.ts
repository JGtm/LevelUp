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
import { skullPresenceAt, skullSocle } from '../model/skullPresence'

import { useCarrierPosAt } from '../model/carrierPosition'
import { type CanvasView } from '../model/replayView'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

interface UseReplayObjectiveObjectsArgs {
  /**
   * LE DOCUMENT ENTIER, et non les deux listes séparément, depuis le lot 6.7 phase B2 : la
   * présence du crâne dépend désormais aussi des TRAJECTOIRES de son porteur (un portage dont le
   * porteur n'a pas de position rend le crâne LIBRE, cf. `skullPresence.ts`), et cette relecture
   * est celle du résolveur commun — le même que les quatre autres calques de porteur.
   */
  doc: ReplayDocumentReady
  view: CanvasView
  /** L'encre neutre du thème (remplissage) et celle du FOND (liseré), déjà résolues par l'appelant. */
  ink: string
  outline: string
}

export interface ReplayObjectiveObjects {
  /**
   * LE NOM DU CALQUE, porté par le calque et non par la table de liaison du canvas
   * (2026-09-06, revue R1 constat C2) : c'est ce qui rend impossible de peindre ce geste
   * sous l'identité d'un autre calque.
   */
  id: 'objets-objectif'
  /** Peint le crâne libre si sa présence est `free` à cette image. No-op sinon (porté / absent). */
  paint: (ctx: CanvasRenderingContext2D, frame: number) => void
}

export function useReplayObjectiveObjects({
  doc, view, ink, outline,
}: UseReplayObjectiveObjectsArgs): ReplayObjectiveObjects {
  const lives = doc.objectiveObjects
  const carries = doc.skullCarries
  const layer = useMemo<ObjectiveObjectsInput>(() => ({ style: { ink, outline } }), [ink, outline])
  // Le socle (point de réapparition) se lit UNE fois : le crâne au repos y est posé (cf. skullPresence).
  const socle = useMemo(() => skullSocle(lives), [lives])
  // La relecture de position partagée (carrierPosition.ts) : embarqué -> position du véhicule,
  // sinon celle du bipède. C'est la MÊME que celle de `skullCarrierLayer` — les deux calques
  // doivent s'accorder à l'image près, sinon le crâne clignote entre porté et libre.
  const posOf = useCarrierPosAt(doc)
  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number) => {
      // Sans aucune vie émise, on ne connaît ni position ni socle : rien à peindre.
      if (lives.length === 0) return
      const presence = skullPresenceAt(lives, carries, frame, socle, posOf)
      if (presence.state !== 'free') return
      drawFreeSkull(ctx, layer, presence.at, view, presence.rolling)
    },
    [lives, carries, socle, posOf, layer, view],
  )
  return { id: 'objets-objectif', paint }
}

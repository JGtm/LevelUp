/**
 * livesPosition.ts — L'UNIQUE écriture du bloc « position d'un joueur à une image ».
 *
 * CE QUE CE MODULE CENTRALISE, ET POURQUOI MAINTENANT. Relire la position d'un joueur à une
 * image donnée demande trois pièces : l'index des vies par joueur (`livesByXuid`), la fenêtre
 * après-mort en images (`deathFrames`, dérivée de `KILLPOS_WINDOW_MS`), et la relecture
 * elle-même (`posOfPlayerAt`, killFx.ts — qui reste l'unique écriture de la RELECTURE). Ce
 * bloc était copié dans QUATRE hooks (déflagration, drapeau, crâne, couronne VIP) et deux
 * constructeurs purs (killFx, objectivesLayer) ; le calque du porteur de bombe en aurait été
 * la CINQUIÈME copie. Règle n° 6 du dépôt : à la troisième on centralise — ici on solde tout,
 * et le garde-rail (`livesPosition.guard.test.ts`) interdit la re-divergence.
 *
 * LA FENÊTRE APRÈS-MORT fait partie du bloc, pas une option : un poseur de bombe meurt souvent
 * DANS son explosion, un porteur lâche à sa mort — sans elle, l'événement le mieux daté du
 * match perdrait sa position.
 */

import { useCallback, useMemo } from 'react'

import { KILLPOS_WINDOW_MS, posOfPlayerAt } from './killFx'
import { msToFrames } from './replayLogic'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'

/** La position relue, telle que `posOfPlayerAt` la rend. */
export type PlayerPosAt = (xuid: string, frame: number) => ReturnType<typeof posOfPlayerAt>

/** buildLivesByXuid — l'index des vies par joueur, construit UNE fois par document. */
export function buildLivesByXuid(tracks: readonly ReplayTrackReady[]): Map<string, ReplayTrackReady[]> {
  const map = new Map<string, ReplayTrackReady[]>()
  for (const t of tracks) {
    if (!t.xuid) continue
    const list = map.get(t.xuid)
    if (list) list.push(t)
    else map.set(t.xuid, [t])
  }
  return map
}

/** deathWindowFrames — la fenêtre après-mort en images (au moins une). */
export function deathWindowFrames(doc: ReplayDocumentReady): number {
  return Math.max(1, Math.round(msToFrames(KILLPOS_WINDOW_MS, doc)))
}

/** buildPlayerPosAt — la relecture pure, pour les constructeurs hors React. */
export function buildPlayerPosAt(doc: ReplayDocumentReady): PlayerPosAt {
  const lives = buildLivesByXuid(doc.tracks)
  const window = deathWindowFrames(doc)
  return (xuid, frame) => posOfPlayerAt(lives.get(xuid), frame, window)
}

/** usePlayerPosAt — la même relecture, mémoïsée pour les hooks de calque. */
export function usePlayerPosAt(doc: ReplayDocumentReady): PlayerPosAt {
  const lives = useMemo(() => buildLivesByXuid(doc.tracks), [doc.tracks])
  const window = useMemo(() => deathWindowFrames(doc), [doc])
  return useCallback<PlayerPosAt>(
    (xuid, frame) => posOfPlayerAt(lives.get(xuid), frame, window),
    [lives, window],
  )
}

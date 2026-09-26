/**
 * equipmentUsageFixtures.ts — LE DÉCOR PARTAGÉ des tests du bilan d'équipement.
 *
 * POURQUOI CE FICHIER (CLAUDE.md n°6, « à la 3e copie on centralise »). `equipmentUsageLogic.
 * test.ts` et `equipmentUsageLogic.kept.test.ts` (extrait le 2026-09-09, seuil de taille du
 * dépôt) partagent le MÊME témoin — quatre joueurs, deux camps, un slot caméra sans
 * propriétaire — et la même construction de pose. Deux copies auraient divergé au premier
 * ajustement de frame ; un témoin partagé garde les deux fichiers d'accord sur ce qu'ils
 * mesurent (même patron que `placementFixtures.ts`).
 */
import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { testReplayDoc } from './testDoc'

/** Une vie : le slot, son propriétaire, et deux points pour que la fenêtre existe. */
export function vie(slot: number, xuid: string, start = 0, end = 100) {
  return {
    slot,
    xuid,
    team: -1,
    startFrame: start,
    endFrame: end,
    points: [
      { t: start, x: 0, y: 0 },
      { t: end, x: 1, y: 1 },
    ],
  }
}

/** Une pose d'équipement (les champs que l'agrégation lit). */
export function pose(family: string, origin: string, owner: number, id = '0xaaaa') {
  return { family, origin, owner, id, t0: 10, t1: 20, x: 0, y: 0 }
}

export const SB: MatchScoreboardRow[] = [
  { xuid: 'a1', gamertag: 'Alpha', team_side: 't0' },
  { xuid: 'a2', gamertag: 'Bravo', team_side: 't0' },
  { xuid: 'b1', gamertag: 'Charlie', team_side: 't1' },
] as MatchScoreboardRow[]

/**
 * LE TÉMOIN. Trois joueurs au scoreboard (deux camps) plus un QUATRIÈME que le film voit vivre
 * et que le scoreboard ignore ; un slot de caméra (vie sans xuid) qui porte pourtant des gestes.
 *
 * Frames à 100 ms : un épisode de 50 frames dure 5 000 ms — les durées sont donc lisibles à
 * l'œil dans les attentes des tests qui l'emploient.
 */
export function temoin(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
      { filmIndex: 1, xuid: 'a2', name: 'Bravo' },
      { filmIndex: 2, xuid: 'b1', name: 'Charlie' },
      { filmIndex: 3, xuid: 'orphelin', name: 'Delta' },
    ],
    tracks: [
      vie(1, 'a1'),
      vie(2, 'a2'),
      vie(3, 'b1'),
      vie(4, 'orphelin'),
      // Une vie SANS propriétaire : caméra ou spectateur de fin de partie.
      { slot: 9, team: -1, startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }] },
    ],
    ...over,
  } as Partial<ReplayDocument>)
}

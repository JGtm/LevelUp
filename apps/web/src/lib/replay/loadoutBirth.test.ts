/**
 * Tests — loadoutAt et la DOTATION DE NAISSANCE (schéma 69, lot M3.3 de la campagne « retours
 * rejeu », 2026-09-23). Séparés de `rosterLogic.test.ts`, qui atteignait le seuil de taille.
 *
 * CE QUE CE FICHIER VERROUILLE : la dotation de naissance est le premier relevé PASSÉ de la vie,
 * avec sa provenance et ses emplacements ; la lecture « à venir » disparaît d'une vie dont la
 * naissance est lue (décision Q18) et reste pour une vie sans dotation, ou un artefact antérieur.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayTrackReady } from './replayNormalize'
import { testReplayDoc as doc } from '../../features/match-replay/test/testDoc'
import { loadoutAt } from './rosterLogic'

/** Une vie [start, end] sur un slot : deux points suffisent à la borner. */
function track(slot: number, xuid: string, start: number, end: number): ReplayTrackReady {
  return {
    slot,
    team: -1,
    xuid,
    startFrame: start,
    endFrame: end,
    points: [
      { t: start, x: 0, y: 0, sh: 1 },
      { t: end, x: 1, y: 1 },
    ],
  }
}

describe('loadoutAt — la dotation de NAISSANCE (schéma 69, lot M3.3)', () => {
  // Une vie [0,250] ; sa naissance est lue à t=3 (deux armes, emplacements 0 et 1), sa première
  // image-clé à t=200. Le bloc `coverage.birthLoadouts` dit que le producteur a lu les naissances.
  const naissance = { t: 3, slot: 512, w: ['0xAAAA', '0xBBBB'], src: 'birth', k: [0, 1] }
  const lue = doc({
    tracks: [track(512, 'A', 0, 250)],
    loadouts: [naissance, { t: 200, slot: 512, w: ['0xCCCC'] }],
    coverage: { birthLoadouts: { creations: 1, read: 1 } } as never,
  })

  it('la dotation de naissance est le premier relevé PASSÉ de la vie, avec sa provenance', () => {
    expect(loadoutAt(lue, 512, 60)).toEqual({ weapons: ['0xAAAA', '0xBBBB'], age: 57, src: 'birth', k: [0, 1] })
  })

  it('Q18 : plus AUCUNE lecture à venir sur une vie dont la naissance est lue', () => {
    // frame 1 : la naissance (t=3) n'a pas encore eu lieu — « armes non lues », jamais l'avenir.
    expect(loadoutAt(lue, 512, 1)).toBeNull()
  })

  it('une vie SANS dotation garde le repli à venir, même dans un document qui lit les naissances', () => {
    // Le slot 513 n'a pas de naissance lue (record non fermé, build ancien) : sans le repli, sa
    // fiche resterait « armes non lues » jusqu'à la première image-clé.
    const mixte = doc({
      tracks: [track(512, 'A', 0, 250), track(513, 'B', 0, 250)],
      loadouts: [naissance, { t: 200, slot: 513, w: ['0xDDDD'] }],
      coverage: { birthLoadouts: { creations: 2, read: 1 } } as never,
    })
    expect(loadoutAt(mixte, 513, 60)).toEqual({ weapons: ['0xDDDD'], age: -140 })
    expect(loadoutAt(mixte, 512, 1)).toBeNull()
  })

  it('un artefact ANTÉRIEUR (sans le bloc) garde le repli à venir', () => {
    const ancien = doc({ tracks: [track(512, 'A', 0, 250)], loadouts: [{ t: 200, slot: 512, w: ['0xCCCC'] }] })
    expect(loadoutAt(ancien, 512, 60)).toEqual({ weapons: ['0xCCCC'], age: -140 })
  })
})

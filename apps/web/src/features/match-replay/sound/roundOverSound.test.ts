/**
 * roundOverSound.test.ts — LE SON « MANCHE TERMINÉE ».
 *
 * CE QU'IL PROTÈGE : un son PAR bascule de manche, à la frame de la bascule (la même que
 * l'overlay inter-manche), dans la langue de l'interface ; rien sur un mode à manche unique ;
 * rien quand l'horloge du film n'est pas recalée (garde de `scoreTimelineOf`).
 */
import { describe, expect, it } from 'vitest'

import { normalizeScoreTimeline } from '@/lib/replay/scoreTimeline'

import type { ReplayDocumentReady } from '../replayNormalize'
import { ROUND_OVER_SOUND_STEMS, roundOverSoundEvents } from './roundOverSound'

/** Une manche : numéro et paliers `[frame, valeur]`. */
function manche(round: number, points: Array<[number, number]>) {
  return { round, points: points.map(([t, v]) => ({ t, v })) }
}

/** Oddball à deux manches : la bascule tombe à la FIN de la manche 1 (frame 50). */
const ODDBALL_TEAMS = [
  {
    teamId: 0,
    rounds: [manche(0, [[0, 0], [50, 100]]), manche(1, [[100, 0], [150, 100]])],
    total: [[0, 0], [50, 100], [150, 200]].map(([t, v]) => ({ t, v })),
  },
  {
    teamId: 1,
    rounds: [manche(0, [[0, 0], [50, 78]]), manche(1, [[100, 0], [150, 43]])],
    total: [[0, 0], [50, 78], [150, 121]].map(([t, v]) => ({ t, v })),
  },
]

/** Un document réduit à ce que la piste demande. `frameIntervalMs: 100` → la frame 50 = 5 000 ms. */
function docOf(teams: unknown[], extra: Record<string, unknown> = {}): ReplayDocumentReady {
  return {
    frameIntervalMs: 100,
    originMs: 0,
    scoreTimeline: normalizeScoreTimeline({ teams, players: [] } as never),
    ...extra,
  } as unknown as ReplayDocumentReady
}

describe('roundOverSoundEvents — quand et quoi', () => {
  it('un son à la bascule, à la frame de FIN de la manche qui se termine', () => {
    const evs = roundOverSoundEvents(docOf(ODDBALL_TEAMS), 'fr')
    expect(evs).toHaveLength(1)
    // LA FIN DE LA MANCHE, PAS LA REPRISE (correctif du 2026-08-29) : la voix d'annonceur
    // partait au premier point de la manche suivante, 19 à 34 s trop tard sur les témoins.
    expect(evs[0].ms).toBe(5_000) // frame 50 × frameIntervalMs 100
  })

  it('joue le stem de la LANGUE de l interface', () => {
    expect(roundOverSoundEvents(docOf(ODDBALL_TEAMS), 'fr')[0].stem).toBe(ROUND_OVER_SOUND_STEMS.fr)
    expect(roundOverSoundEvents(docOf(ODDBALL_TEAMS), 'en')[0].stem).toBe(ROUND_OVER_SOUND_STEMS.en)
  })

  it('rien sur un mode à manche unique (aucune bascule)', () => {
    const slayer = [
      { teamId: 0, rounds: [manche(0, [[0, 0], [500, 43]])], total: [{ t: 0, v: 0 }, { t: 500, v: 43 }] },
    ]
    expect(roundOverSoundEvents(docOf(slayer), 'fr')).toEqual([])
  })

  it('rien quand l horloge du film n est pas recalée (garde de scoreTimelineOf)', () => {
    const evs = roundOverSoundEvents(
      docOf(ODDBALL_TEAMS, { originMs: undefined, coverage: { originResolved: false } }),
      'fr',
    )
    expect(evs).toEqual([])
  })
})

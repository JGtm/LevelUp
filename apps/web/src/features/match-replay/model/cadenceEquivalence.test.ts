/**
 * cadenceEquivalence.test.ts — LA LECTURE DU REJEU NE DÉPEND PAS DU PAS DE LA GRILLE.
 *
 * Chaque cas pose les MÊMES instants en millisecondes sur une grille de 100 ms et sur une grille
 * de 33 ms (30,3 images/s), les convertit en images avec le pas du document, et exige la même
 * décision : mêmes bandes de la frise, même focus d'élimination, même fenêtre de la jauge de
 * colline, même traînée de projectile. Un test qui ne couvre que 100 ms laisserait passer un seuil
 * recopié en images.
 */
import { describe, expect, it } from 'vitest'

import { frameToMs, msToFrames } from '@/lib/replay/replayLogic'
import type { ReplayProjectileReady } from '@/lib/replay/replayNormalize'
import type { ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

import { PROJECTILE_TAIL_MS, drawProjectilesLayer } from '../layers/replayProjectiles'
import { recordingContext } from '../test/recordingContext'
import { HOLD_FULL_WINDOW_MS, readHillHold } from './hillHoldLogic'
import { frameAtOrAfter } from './killFocusLogic'
import { buildFragDominance } from './replayTimelineTracksLogic'

const PAS = [100, 33] as const

describe('frameAtOrAfter — la première image qui ne précède pas l’instant', () => {
  for (const pas of PAS) {
    it(`pas de ${pas} ms : l'image rendue est au plus à un pas après l'instant, jamais avant`, () => {
      const doc = { frameIntervalMs: pas }
      for (const ms of [0, 1, 99, 100, 1234, 41_000, 504_300]) {
        const frame = frameAtOrAfter(ms, doc)
        expect(frameToMs(frame, doc), `${ms} ms`).toBeGreaterThanOrEqual(ms)
        expect(frameToMs(frame, doc) - ms, `${ms} ms`).toBeLessThan(pas)
      }
    })
  }
})

describe('buildFragDominance — les bandes de la frise sont les mêmes aux deux pas', () => {
  const FRAGS = [
    { replayMs: 2_000, teamId: 0 },
    { replayMs: 5_000, teamId: 1 },
    { replayMs: 5_000, teamId: 1 },
    { replayMs: 9_000, teamId: 0 },
  ]
  const DUREE_MS = 12_000

  function bandes(pas: number) {
    const span = DUREE_MS / pas
    return buildFragDominance(FRAGS, pas, { from: 0, span })
  }

  it('mêmes camps dans le même ordre, mêmes bornes à un millième près', () => {
    const ref = bandes(100)
    const autre = bandes(33)
    expect(autre.map((s) => s.teamId)).toEqual(ref.map((s) => s.teamId))
    expect(autre.map((s) => s.key)).toEqual(ref.map((s) => s.key))
    for (const [i, s] of ref.entries()) {
      expect(autre[i].from).toBeCloseTo(s.from, 9)
      expect(autre[i].to).toBeCloseTo(s.to, 9)
    }
  })
})

describe('readHillHold — la fenêtre de pleine jauge est une durée', () => {
  const REMISE_MS = 3_300 // multiple de 100 et de 33 : la même image exacte à 100 ms et 33 ms par 33 ms

  function documentAuPas(pas: number) {
    const frame = (ms: number) => Math.round(ms / pas)
    return {
      frameIntervalMs: pas,
      scoreTimeline: {
        players: [],
        teams: [
          { teamId: 0, rounds: [], total: [{ t: frame(REMISE_MS), v: 1 }] },
          { teamId: 1, rounds: [], total: [] },
        ],
        holdTicks: [
          { teamId: 0, ticks: [{ t: frame(REMISE_MS - 600), v: 20 }, { t: frame(REMISE_MS), v: 35 }, { t: frame(7_000), v: 45 }] },
          { teamId: 1, ticks: [{ t: frame(REMISE_MS + 3_000), v: 5 }] },
        ],
        holdTicksPerPoint: 35,
      } as ReplayScoreTimelineReady,
    }
  }

  for (const pas of PAS) {
    it(`pas de ${pas} ms : la jauge du camp qui marque est pleine pendant ${HOLD_FULL_WINDOW_MS} ms, puis repart de zéro`, () => {
      const doc = documentAuPas(pas)
      const frame = (ms: number) => Math.floor(ms / pas)
      expect(readHillHold(doc, 0, 1, frame(REMISE_MS + 900))!.ally).toBe(1)
      const apres = readHillHold(doc, 0, 1, frame(REMISE_MS + 1_200))!
      expect(apres.ally).toBeCloseTo(0, 5)
      expect(readHillHold(doc, 0, 1, frame(7_000))!.ally).toBeCloseTo(10 / 35, 5)
      expect(readHillHold(doc, 0, 1, frame(7_000))!.enemy).toBeCloseTo(5 / 35, 5)
    })
  }
})

describe('drawProjectilesLayer — la traînée d’un vol est une durée', () => {
  const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 100, height: 100, pad: 0 }
  const VOL_MS = 1_000

  function traces(pas: number, apresLeDernierPointMs: number): number {
    const doc = { frameIntervalMs: pas }
    const t0 = 10
    const dernier = Math.round(VOL_MS / pas)
    const vol = { t0, p: [[0, 1, 1], [dernier, 4, 4]], detoT: t0 + dernier, detoX: 4, detoY: 4 } as ReplayProjectileReady
    const { ctx, ops } = recordingContext()
    drawProjectilesLayer(
      ctx,
      [vol],
      VIEW,
      { frame: t0 + dernier + Math.floor(apresLeDernierPointMs / pas), tailFrames: msToFrames(PROJECTILE_TAIL_MS, doc) },
      '#c',
    )
    return ops.filter((o) => o.op === 'stroke').length
  }

  for (const pas of PAS) {
    it(`pas de ${pas} ms : visible 600 ms après le dernier point, effacé 800 ms après`, () => {
      expect(traces(pas, 600)).toBe(1)
      expect(traces(pas, 800)).toBe(0)
    })
  }
})

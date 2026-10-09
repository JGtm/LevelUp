/**
 * Tests — la COLLINE au calque des zones (schéma 92) : une seule colline visible, sa jauge de
 * capture (prise et vidange), son étage.
 *
 * CE QU'ILS PROTÈGENT : le calque statique ne dessine plus les collines d'un document à collines
 * (`staticObjectivesOf`) et garde tout le reste ; une colline ne se peint que pendant ses
 * intervalles, jauge comprise ; la colline active porte ses contours d'étage (le langage des
 * autres objectifs) ; la jauge qui SE VIDE prend l'encre du camp qui tient encore la colline, la
 * prise celle du camp qui pousse.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayMapObjectives } from '@/lib/api/types'

import { normalizeMapObjectives, OBJECTIVE_TEAM_NEUTRAL } from './objectivesLayer'
import { count, recordingContext, type CanvasOp } from '../test/recordingContext'
import type { ReplayZoneStateReady } from '../../../lib/replay/replayNormalize'
import { drawZoneStates, staticObjectivesOf, zoneElementsOf } from './zoneStatesLayer'

const MO: ReplayMapObjectives = {
  zones: [
    { role: 'hill', team: OBJECTIVE_TEAM_NEUTRAL, x: 3, y: 3, z: 9, family: 'cylinder', radius: 2, fwdX: 1, fwdY: 0 },
    { role: 'hill', team: OBJECTIVE_TEAM_NEUTRAL, x: 7, y: 7, z: 1, family: 'cylinder', radius: 2, fwdX: 1, fwdY: 0 },
  ],
  markers: [{ role: 'flag_spawn', team: 0, x: 1, y: 9, z: 1 }],
}
const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 528, height: 528, pad: 24 }
const HOLD = 10
const style = {
  colorOfOwner: (team: number) => (team === 0 ? '#allié' : '#adverse'),
  colorOfCapturer: (team: number) => (team === 0 ? '#allié' : '#adverse'),
  neutral: '#neutre',
}
const layer = (z = { min: 0, max: 10 }) => ({
  zoneElements: zoneElementsOf(normalizeMapObjectives(MO)),
  joinable: true,
  style,
  gaugeHoldFrames: HOLD,
  z,
})

/**
 * Deux collines : la 0 active de 10 à 49 (prise par le camp 1 à 20, puis vidée de 30 à 39 et
 * neutre à 40), la 1 active de 50 à 99.
 */
const HILLS: ReplayZoneStateReady[] = [
  {
    zoneRef: 0,
    spans: [
      { t0: 10, t1: 19, owner: null, active: true },
      { t0: 20, t1: 39, owner: 1, active: true },
      { t0: 40, t1: 49, owner: null, active: true },
    ],
    gauge: [
      { t: 10, v: 0.1 }, { t: 15, v: 0.6 }, { t: 19, v: 0.97 }, { t: 20, v: 0 },
      { t: 30, v: 0.98 }, { t: 35, v: 0.5 }, { t: 39, v: 0.1 }, { t: 40, v: 0 },
    ],
    gaugeRamps: [
      { t0: 10, t1: 20, capturingTeam: 1 },
      { t0: 30, t1: 40, draining: true },
    ],
  },
  { zoneRef: 1, spans: [{ t0: 50, t1: 99, owner: 0, active: true }], gauge: [], gaugeRamps: [] },
]

/** L'encre en vigueur au moment du dernier remplissage de progression. */
const encreDeLaProgression = (ops: CanvasOp[]): string | undefined => {
  const dernier = ops.map((o) => o.op).lastIndexOf('fillRect')
  if (dernier < 0) return undefined
  const avant = ops.slice(0, dernier).filter((o) => o.op === 'set fillStyle')
  return avant.length > 0 ? String(avant[avant.length - 1].args[0]) : undefined
}

describe('staticObjectivesOf — le calque statique ne dessine plus les collines', () => {
  const elements = normalizeMapObjectives(MO)
  it('document à collines joignable : les zones sortent, les marqueurs restent', () => {
    const out = staticObjectivesOf(elements, HILLS, true)
    expect(out.filter((e) => e.kind === 'zone')).toHaveLength(0)
    expect(out.filter((e) => e.kind === 'marker')).toHaveLength(1)
  })
  it('zones simultanées (aucun intervalle actif) : rien ne change', () => {
    const bases = [{ spans: [{ t0: 0, t1: 9, active: false }] }]
    expect(staticObjectivesOf(elements, bases, true)).toBe(elements)
  })
  it('jointure douteuse : le statique reste seul, il garde ses zones', () => {
    expect(staticObjectivesOf(elements, HILLS, false)).toBe(elements)
  })
})

describe('drawZoneStates — la colline', () => {
  it('une seule colline peinte à la fois, et rien hors de ses intervalles', () => {
    for (const [frame, attendu] of [[5, 0], [25, 1], [60, 1], [120, 0]] as const) {
      const { ctx, ops } = recordingContext()
      drawZoneStates(ctx, layer(), HILLS, VIEW, frame)
      expect(count(ops, 'fill') > 0 ? 1 : 0, `frame ${frame}`).toBe(attendu)
    }
  })

  it('la jauge d une colline inactive ne se peint pas, même tenue par l escalier', () => {
    const hors: ReplayZoneStateReady = { ...HILLS[0], spans: [{ t0: 40, t1: 49, owner: null, active: true }] }
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), [hors], VIEW, 15)
    expect(count(ops, 'fillRect')).toBe(0)
  })

  it('la prise se remplit à l encre du camp qui pousse', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), HILLS, VIEW, 15)
    expect(count(ops, 'fillRect')).toBe(1)
    expect(encreDeLaProgression(ops)).toBe('#adverse')
  })

  it('la vidange se remplit à l encre du camp qui tient encore la colline', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), HILLS, VIEW, 35)
    expect(count(ops, 'fillRect')).toBe(1)
    expect(encreDeLaProgression(ops)).toBe('#adverse')
    const libre = recordingContext()
    const tenueParZero: ReplayZoneStateReady = {
      ...HILLS[0],
      spans: [{ t0: 10, t1: 49, owner: 0, active: true }],
    }
    drawZoneStates(libre.ctx, layer(), [tenueParZero], VIEW, 35)
    expect(encreDeLaProgression(libre.ops)).toBe('#allié')
  })

  it('la colline active porte ses contours d étage ; au sol, aucun', () => {
    const haute = recordingContext()
    drawZoneStates(haute.ctx, layer(), HILLS, VIEW, 25) // colline 0 : z 9 sur [0 ; 10], étage 2
    const basse = recordingContext()
    drawZoneStates(basse.ctx, layer(), HILLS, VIEW, 60) // colline 1 : z 1, étage 0
    expect(count(haute.ops, 'stroke') - count(basse.ops, 'stroke')).toBe(2)
  })
})

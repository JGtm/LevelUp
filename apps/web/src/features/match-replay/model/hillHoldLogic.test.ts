import { describe, expect, it } from 'vitest'

import type { ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

import { readHillHold, type HillHoldDocument } from './hillHoldLogic'

type Tick = { t: number; v: number }

/**
 * Un calque de score minimal : les paliers de points par camp (qui donnent les remises à zéro),
 * les séries de tics de garde, et le dénominateur.
 */
function doc(over: {
  points?: Record<number, Tick[]>
  hold?: Record<number, Tick[]>
  perPoint?: number
}): HillHoldDocument {
  const points = over.points ?? { 0: [], 1: [] }
  const hold = over.hold ?? { 0: [], 1: [] }
  return {
    scoreTimeline: {
      players: [],
      teams: Object.entries(points).map(([id, total]) => ({ teamId: Number(id), rounds: [], total })),
      holdTicks: Object.entries(hold).map(([id, ticks]) => ({ teamId: Number(id), ticks })),
      ...(over.perPoint === undefined ? {} : { holdTicksPerPoint: over.perPoint }),
    } as ReplayScoreTimelineReady,
  }
}

/** 35 tics = un point : la valeur mesurée de la variante. */
const PER_POINT = 35

describe('readHillHold — les cas où la jauge NE DOIT PAS se dessiner', () => {
  it('se tait sans dénominateur mesuré — le cas du KOTH SQUAD, jamais une valeur devinée', () => {
    expect(readHillHold(doc({}), 0, 1, 500)).toBeNull()
  })

  it('se tait sans série de garde (mode sans colline, ou artefact antérieur au champ)', () => {
    expect(readHillHold({ scoreTimeline: undefined }, 0, 1, 500)).toBeNull()
    expect(readHillHold(doc({ hold: {}, perPoint: PER_POINT }), 0, 1, 500)).toBeNull()
  })

  it("se tait quand un camp n'est pas situé par le calque", () => {
    const d = doc({ hold: { 0: [{ t: 10, v: 1 }] }, perPoint: PER_POINT })
    expect(readHillHold(d, 0, 1, 500)).toBeNull()
  })
})

describe('readHillHold — la progression', () => {
  it('rapporte les tics pris au dénominateur de la variante', () => {
    const d = doc({
      hold: {
        0: [
          { t: 100, v: 7 },
          { t: 200, v: 14 },
        ],
        1: [{ t: 150, v: 3 }],
      },
      perPoint: PER_POINT,
    })
    const r = readHillHold(d, 0, 1, 250)
    expect(r).not.toBeNull()
    expect(r!.ally).toBeCloseTo(14 / 35, 5)
    expect(r!.enemy).toBeCloseTo(3 / 35, 5)
  })

  it("lit la série EN ESCALIER : la dernière valeur tient jusqu'au point suivant", () => {
    const d = doc({
      hold: {
        0: [
          { t: 100, v: 7 },
          { t: 400, v: 20 },
        ],
        1: [],
      },
      perPoint: PER_POINT,
    })
    expect(readHillHold(d, 0, 1, 399)!.ally).toBeCloseTo(7 / 35, 5)
    expect(readHillHold(d, 0, 1, 400)!.ally).toBeCloseTo(20 / 35, 5)
  })

  it('remet les DEUX jauges à zéro au point marqué, et lit la suite en différentiel', () => {
    const d = doc({
      points: { 0: [{ t: 300, v: 1 }], 1: [] },
      hold: {
        0: [
          { t: 100, v: 20 },
          { t: 300, v: 35 },
          { t: 500, v: 45 },
        ],
        1: [{ t: 400, v: 5 }],
      },
      perPoint: PER_POINT,
    })
    // Juste avant le point : ce que le camp 0 a accumulé.
    expect(readHillHold(d, 0, 1, 299)!.ally).toBeCloseTo(20 / 35, 5)
    // Après : 45 − 35 = 10 tics depuis la remise à zéro, pas 45.
    expect(readHillHold(d, 0, 1, 500)!.ally).toBeCloseTo(10 / 35, 5)
    // Et l'adversaire repart de zéro lui aussi : ses 5 tics sont tous postérieurs au point.
    expect(readHillHold(d, 0, 1, 500)!.enemy).toBeCloseTo(5 / 35, 5)
  })

  it('ignore les RÉ-ÉMISSIONS de la même valeur : seul un palier plus haut est un point', () => {
    const d = doc({
      points: {
        0: [
          { t: 100, v: 1 },
          { t: 300, v: 1 },
        ],
        1: [],
      },
      hold: {
        0: [
          { t: 100, v: 35 },
          { t: 500, v: 55 },
        ],
        1: [],
      },
      perPoint: PER_POINT,
    })
    // La remise à zéro est à l'image 100, pas à 300 : 55 − 35 = 20.
    expect(readHillHold(d, 0, 1, 500)!.ally).toBeCloseTo(20 / 35, 5)
  })

  it('CLAMPE à 1 — une barre au-delà se lirait comme un défaut de rendu', () => {
    const d = doc({ hold: { 0: [{ t: 100, v: 90 }], 1: [] }, perPoint: PER_POINT })
    expect(readHillHold(d, 0, 1, 200)!.ally).toBe(1)
  })

  it("ne décroît jamais à l'intérieur d'une période", () => {
    const d = doc({
      hold: {
        0: [
          { t: 100, v: 5 },
          { t: 200, v: 12 },
          { t: 300, v: 30 },
        ],
        1: [],
      },
      perPoint: PER_POINT,
    })
    let prev = -1
    for (let f = 0; f <= 400; f += 25) {
      const v = readHillHold(d, 0, 1, f)!.ally
      expect(v).toBeGreaterThanOrEqual(prev)
      prev = v
    }
  })
})

describe('readHillHold — la mécanique de garde énoncée par l utilisateur (2026-10-09)', () => {
  // Le camp 0 tient la colline (10 tics à la frame 100), la perd au camp 1 (5 tics de 150 à 200),
  // la reprend (20 tics à 300), puis marque (35 à 400) : la barre du camp 0 NE DESCEND PAS
  // pendant la perte, REPREND où elle en était, et les deux barres se VIDENT au point.
  const d = doc({
    points: { 0: [{ t: 400, v: 1 }], 1: [] },
    hold: {
      0: [{ t: 100, v: 10 }, { t: 300, v: 20 }, { t: 400, v: 35 }],
      1: [{ t: 150, v: 2 }, { t: 200, v: 5 }],
    },
    perPoint: PER_POINT,
  })
  it('ne descend pas quand la colline est perdue', () => {
    expect(readHillHold(d, 0, 1, 120)!.ally).toBeCloseTo(10 / 35)
    expect(readHillHold(d, 0, 1, 250)!.ally).toBeCloseTo(10 / 35)
    expect(readHillHold(d, 0, 1, 250)!.enemy).toBeCloseTo(5 / 35)
  })
  it('reprend où elle en était quand le camp reprend la colline', () => {
    expect(readHillHold(d, 0, 1, 350)!.ally).toBeCloseTo(20 / 35)
  })
  it('se vide pour les deux camps au point marqué', () => {
    expect(readHillHold(d, 0, 1, 400)).toEqual({ ally: 0, enemy: 0 })
    expect(readHillHold(d, 0, 1, 450)).toEqual({ ally: 0, enemy: 0 })
  })
})

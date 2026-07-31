import { describe, expect, it } from 'vitest'

import type { ReplayDocument, ReplayTrack } from '@/lib/api/types'

import {
  advanceFrame,
  altitudeAt,
  altitudeRatio,
  fitWidth,
  floorOf,
  footprint,
  formatClock,
  frameToMs,
  framesPerSecond,
  freshness,
  heldReading,
  isAliveAt,
  lastIndexAt,
  msToFrames,
  positionAt,
  sceneBounds,
  trackWindow,
  trailAt,
  worldToCanvas,
} from './replayLogic'

const pts = [
  { t: 0, x: 0, y: 0 },
  { t: 10, x: 10, y: 20 },
  { t: 20, x: 10, y: 20 },
]

function makeDoc(over: Partial<ReplayDocument> = {}): ReplayDocument {
  return {
    schemaVersion: 1,
    matchId: 'm',
    titleSlug: 'halo_infinite',
    frameCount: 100,
    bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
    tracks: [],
    ...over,
  }
}

describe('positionAt', () => {
  it('null avant le 1er point', () => expect(positionAt(pts, -1)).toBeNull())
  it('exact au 1er point', () => expect(positionAt(pts, 0)).toEqual({ x: 0, y: 0 }))
  it('interpole entre deux points', () => expect(positionAt(pts, 5)).toEqual({ x: 5, y: 10 }))
  it('maintient la dernière position après la fin', () =>
    expect(positionAt(pts, 99)).toEqual({ x: 10, y: 20 }))
  it('liste vide -> null', () => expect(positionAt([], 3)).toBeNull())
})

describe('advanceFrame', () => {
  it('avance de deltaFrames', () => expect(advanceFrame(0, 2, 100)).toBe(2))
  it('boucle à 0 en fin de rejeu', () => expect(advanceFrame(98, 5, 100)).toBe(0))
  it('frameCount<=1 -> 0', () => expect(advanceFrame(0, 1, 1)).toBe(0))
  it('ne descend pas sous 0', () => expect(advanceFrame(0, -5, 100)).toBe(0))
})

describe('worldToCanvas', () => {
  const b = { minX: 0, minY: 0, maxX: 10, maxY: 10 }
  it('coin monde haut-gauche (0,10) -> haut-gauche canvas (Y inversé)', () => {
    const c = worldToCanvas({ x: 0, y: 10 }, b, 100, 100, 10)
    expect(c.x).toBeCloseTo(10)
    expect(c.y).toBeCloseTo(10)
  })
  it('coin monde bas-droit (10,0) -> bas-droit canvas', () => {
    const c = worldToCanvas({ x: 10, y: 0 }, b, 100, 100, 10)
    expect(c.x).toBeCloseTo(90)
    expect(c.y).toBeCloseTo(90)
  })
})

describe('trailAt', () => {
  it('borne la traînée à la fenêtre et finit à la tête', () => {
    const tr = trailAt(pts, 10, 10)
    expect(tr[0]).toEqual({ x: 0, y: 0 })
    expect(tr[tr.length - 1]).toEqual({ x: 10, y: 20 })
  })
  it('exclut les points hors fenêtre', () => {
    const tr = trailAt(pts, 20, 5)
    expect(tr.every((p) => p.x === 10 && p.y === 20)).toBe(true)
  })
})

describe('fitWidth', () => {
  const square = { minX: 0, minY: 0, maxX: 10, maxY: 10 }
  it('scène carrée : largeur = hauteur (marges latérales supprimées)', () =>
    expect(fitWidth(square, 900, 480, 24)).toBeCloseTo(480))
  it('ne dépasse jamais la largeur disponible', () =>
    expect(fitWidth({ minX: 0, minY: 0, maxX: 100, maxY: 10 }, 600, 480, 24)).toBe(600))
})

describe('altitudeAt', () => {
  const zpts = [
    { t: 0, x: 0, y: 0, z: 0 },
    { t: 10, x: 0, y: 0, z: 4 },
  ]
  it('interpole le Z', () => expect(altitudeAt(zpts, 5)).toBeCloseTo(2))
  it('z absent -> 0', () => expect(altitudeAt(pts, 5)).toBe(0))
  it('avant le 1er point -> null', () => expect(altitudeAt(zpts, -1)).toBeNull())
})

describe('sceneBounds', () => {
  it('sans géométrie, renvoie les bounds des trajectoires', () => {
    const doc = makeDoc()
    expect(sceneBounds(doc)).toEqual(doc.bounds)
  })
  it('cadre sur l’union trajectoires + fond de carte', () => {
    const doc = makeDoc({ geometryBounds: { minX: -5, minY: 2, maxX: 8, maxY: 30 } })
    expect(sceneBounds(doc)).toMatchObject({ minX: -5, minY: 0, maxX: 10, maxY: 30 })
  })
})

describe('échelle temporelle', () => {
  const doc = makeDoc({ frameIntervalMs: 100 })
  it('1× = 10 frames/s pour un pas de 100 ms', () => expect(framesPerSecond(doc)).toBe(10))
  it('retombe sur la cadence historique sans frameIntervalMs', () =>
    expect(framesPerSecond(makeDoc())).toBe(60))
  it('frameToMs suit le temps réel', () => expect(frameToMs(4985, doc)).toBe(498_500))
  it('msToFrames est l’inverse', () => expect(msToFrames(8000, doc)).toBe(80))
  it('formatClock formate en m:ss', () => {
    expect(formatClock(498_500)).toBe('8:18')
    expect(formatClock(9_000)).toBe('0:09')
    expect(formatClock(-5)).toBe('0:00')
  })
})

describe('fenêtre de vie', () => {
  const track: ReplayTrack = { slot: 1, team: -1, points: pts, startFrame: 5, endFrame: 15 }
  it('lit startFrame/endFrame', () => expect(trackWindow(track)).toEqual({ start: 5, end: 15 }))
  it('champs omitempty absents -> 0 et t du dernier point', () =>
    expect(trackWindow({ slot: 1, team: -1, points: pts })).toEqual({ start: 0, end: 20 }))
  it('masque avant la naissance et après la mort', () => {
    expect(isAliveAt(track, 4)).toBe(false)
    expect(isAliveAt(track, 10)).toBe(true)
    expect(isAliveAt(track, 16)).toBe(false)
  })
})

describe('étages', () => {
  it('altitudeRatio normalise et borne', () => {
    expect(altitudeRatio(0, -4, 4)).toBeCloseTo(0.5)
    expect(altitudeRatio(-99, -4, 4)).toBe(0)
    expect(altitudeRatio(99, -4, 4)).toBe(1)
  })
  it('carte plate -> 0,5 (pas d’étage significatif)', () =>
    expect(altitudeRatio(3, 2, 2)).toBe(0.5))
  it('floorOf découpe en 3 tranches, borne haute incluse dans la dernière', () => {
    expect(floorOf(-4, -4, 8)).toBe(0)
    expect(floorOf(0, -4, 8)).toBe(1)
    expect(floorOf(8, -4, 8)).toBe(2)
  })
})

describe('footprint', () => {
  it('sans emprise -> liste vide (rendu en point)', () =>
    expect(footprint({ typeId: 1, x: 0, y: 0 })).toEqual([]))
  it('rectangle non tourné : 4 coins centrés', () => {
    const c = footprint({ typeId: 1, x: 10, y: 20, dx: 2, dy: 4 })
    expect(c).toHaveLength(4)
    expect(c[0].x).toBeCloseTo(9)
    expect(c[0].y).toBeCloseTo(18)
    expect(c[2].x).toBeCloseTo(11)
    expect(c[2].y).toBeCloseTo(22)
  })
  it('yaw 90° échange largeur et profondeur', () => {
    const c = footprint({ typeId: 1, x: 0, y: 0, dx: 2, dy: 4, yaw: 90 })
    expect(c[0].x).toBeCloseTo(2)
    expect(c[0].y).toBeCloseTo(-1)
  })
})

describe('lastIndexAt', () => {
  const p = [
    { t: 0, x: 0, y: 0 },
    { t: 5, x: 0, y: 0 },
    { t: 9, x: 0, y: 0 },
  ]
  it('rend -1 avant le premier point', () => expect(lastIndexAt(p, -1)).toBe(-1))
  it('rend -1 sur une liste vide', () => expect(lastIndexAt([], 3)).toBe(-1))
  it('rend le dernier point <= t', () => {
    expect(lastIndexAt(p, 0)).toBe(0)
    expect(lastIndexAt(p, 4)).toBe(0)
    expect(lastIndexAt(p, 5)).toBe(1)
    expect(lastIndexAt(p, 100)).toBe(2)
  })
})

describe('heldReading', () => {
  // Le flux est différentiel : seuls certains points portent la mesure.
  const p = [
    { t: 0, x: 0, y: 0, sh: 1 },
    { t: 5, x: 0, y: 0 },
    { t: 8, x: 0, y: 0, sh: 0 },
    { t: 12, x: 0, y: 0 },
  ]
  const sh = (q: { sh?: number }) => q.sh

  it('remonte au dernier point qui PORTE la mesure', () => {
    expect(heldReading(p, 6, sh, 20)).toEqual({ value: 1, age: 6 })
  })

  it('publie un ZÉRO — un bouclier brisé est une valeur, pas une absence', () => {
    // C'est le cas que `if (!v)` effacerait, et c'est le plus utile du champ.
    expect(heldReading(p, 8, sh, 20)).toEqual({ value: 0, age: 0 })
    expect(heldReading(p, 13, sh, 20)).toEqual({ value: 0, age: 5 })
  })

  it('rend null au-delà du maintien : une mesure périmée ne s’affiche pas', () => {
    expect(heldReading(p, 30, sh, 10)).toBeNull()
  })

  it('rend null avant toute mesure', () => {
    expect(heldReading(p, -1, sh, 20)).toBeNull()
    expect(heldReading([{ t: 0, x: 0, y: 0 }], 0, sh, 20)).toBeNull()
  })
})

describe('freshness', () => {
  it('une mesure de l’instant est franche', () => expect(freshness(0, 20, 0.62)).toBe(1))
  it('une mesure au bord du maintien est au plancher', () =>
    expect(freshness(20, 20, 0.62)).toBeCloseTo(0.38))
  it('ne descend jamais sous le plancher', () =>
    expect(freshness(1000, 20, 0.62)).toBeCloseTo(0.38))
  it('un maintien nul ne dégrade rien', () => expect(freshness(5, 0, 0.62)).toBe(1))
})

describe('sceneBounds avec un sol reconstruit', () => {
  it('cadre sur la zone JOUÉE, pas sur la structure qui déborde', () => {
    // La structure d'une carte couvre ±250 m là où les joueurs en parcourent 50 : cadrer sur
    // elle réduirait le terrain à un timbre.
    const doc = makeDoc({
      structure: [{ x0: -200, y0: -200, x1: 200, y1: 200, z: 0, zb: -1 }],
      geometryBounds: { minX: -5, minY: 2, maxX: 8, maxY: 30 },
    })
    expect(sceneBounds(doc)).toEqual(doc.bounds)
  })
})

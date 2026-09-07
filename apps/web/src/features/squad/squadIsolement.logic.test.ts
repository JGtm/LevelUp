import { describe, expect, it } from 'vitest'

import type { SquadIsolementPoint } from '@/lib/api/types'

import {
  medianesNuage,
  opaciteDuPoint,
  OPACITE_ATTENUEE,
  OPACITE_PLEINE,
  pointAttenue,
  quadrantDuPoint,
  tailleDuPoint,
} from './squadIsolement.logic'

function couverture(brut: number, n: number, echantillonFaible: boolean, matchs = 3) {
  return {
    taux: n > 0 ? brut / n : 0,
    brut,
    par_match: matchs > 0 ? brut / matchs : 0,
    n,
    echantillon_faible: echantillonFaible,
  }
}

function point(over: Partial<SquadIsolementPoint> = {}): SquadIsolementPoint {
  return {
    xuid: 'x1',
    gamertag: 'Alice',
    session_label: 'S1',
    morts_examinees: 10,
    morts_isolees: 4,
    part_isolee: couverture(4, 10, true),
    couverture: couverture(6, 10, true),
    ...over,
  } as SquadIsolementPoint
}

describe('medianesNuage', () => {
  it('rend null sans point', () => {
    expect(medianesNuage([])).toBeNull()
  })

  it('médiane impaire : la valeur du milieu', () => {
    const pts = [
      point({ part_isolee: couverture(1, 10, true), couverture: couverture(1, 10, true) }),
      point({ part_isolee: couverture(5, 10, true), couverture: couverture(5, 10, true) }),
      point({ part_isolee: couverture(9, 10, true), couverture: couverture(9, 10, true) }),
    ]
    const med = medianesNuage(pts)
    expect(med?.isolement).toBeCloseTo(0.5)
    expect(med?.couverture).toBeCloseTo(0.5)
  })

  it('médiane paire : la moyenne des deux valeurs centrales', () => {
    const pts = [
      point({ part_isolee: couverture(2, 10, true), couverture: couverture(2, 10, true) }),
      point({ part_isolee: couverture(4, 10, true), couverture: couverture(4, 10, true) }),
    ]
    const med = medianesNuage(pts)
    expect(med?.isolement).toBeCloseTo(0.3)
  })
})

describe('quadrantDuPoint', () => {
  const medianes = { isolement: 0.5, couverture: 0.5 }

  it('proche (< médiane) et couvert (>= médiane) -> procheCouvert', () => {
    const p = point({ part_isolee: couverture(2, 10, true), couverture: couverture(6, 10, true) })
    expect(quadrantDuPoint(p, medianes)).toBe('procheCouvert')
  })

  it('loin (>= médiane) et couvert -> loinCouvert', () => {
    const p = point({ part_isolee: couverture(7, 10, true), couverture: couverture(6, 10, true) })
    expect(quadrantDuPoint(p, medianes)).toBe('loinCouvert')
  })

  it('proche et seul (< médiane de couverture) -> procheSeul', () => {
    const p = point({ part_isolee: couverture(2, 10, true), couverture: couverture(3, 10, true) })
    expect(quadrantDuPoint(p, medianes)).toBe('procheSeul')
  })

  it('loin et sans secours -> loinSansSecours', () => {
    const p = point({ part_isolee: couverture(8, 10, true), couverture: couverture(2, 10, true) })
    expect(quadrantDuPoint(p, medianes)).toBe('loinSansSecours')
  })

  it('pile sur les deux médianes -> jamais le quadrant d\'alerte (arbitrage conservateur)', () => {
    const p = point({ part_isolee: couverture(5, 10, true), couverture: couverture(5, 10, true) })
    expect(quadrantDuPoint(p, medianes)).not.toBe('loinSansSecours')
  })
})

describe('pointAttenue / opaciteDuPoint', () => {
  it('échantillon faible -> atténué', () => {
    const p = point({ part_isolee: couverture(4, 10, true) })
    expect(pointAttenue(p)).toBe(true)
    expect(opaciteDuPoint(p)).toBe(OPACITE_ATTENUEE)
  })

  it('échantillon suffisant -> opacité pleine', () => {
    const p = point({ part_isolee: couverture(12, 40, false) })
    expect(pointAttenue(p)).toBe(false)
    expect(opaciteDuPoint(p)).toBe(OPACITE_PLEINE)
  })

  it("l'atténuation lit part_isolee, pas couverture", () => {
    const p = point({
      part_isolee: couverture(12, 40, false),
      couverture: couverture(2, 5, true),
    })
    expect(pointAttenue(p)).toBe(false)
  })
})

describe('tailleDuPoint', () => {
  it('croît avec les morts examinées', () => {
    expect(tailleDuPoint(20)).toBeGreaterThan(tailleDuPoint(5))
  })

  it('plafonne pour une session très chargée', () => {
    expect(tailleDuPoint(1000)).toBe(tailleDuPoint(500))
  })

  it('reste au-dessus du minimum même à 0 mort', () => {
    expect(tailleDuPoint(0)).toBeGreaterThan(0)
  })
})

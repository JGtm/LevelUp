import { describe, expect, it } from 'vitest'

import type { SquadIsolementPoint } from '@/lib/api/types'

import {
  medianesNuage,
  pointAttenue,
  pointMedianJoueur,
  quadrantDuPoint,
  tailleDuPoint,
  tailleMedianeDuPoint,
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

describe('pointAttenue', () => {
  // Depuis le lot C3 (2026-09-09), l'échantillon faible se signale par un
  // CERCLE POINTILLÉ (SquadIsolementNuageCard), plus par une opacité réduite
  // — `opaciteDuPoint`/`OPACITE_ATTENUEE`/`OPACITE_PLEINE` sont retirés (plus
  // aucun appelant, CLAUDE.md règle 7 : zéro code mort).
  it('échantillon faible -> atténué', () => {
    const p = point({ part_isolee: couverture(4, 10, true) })
    expect(pointAttenue(p)).toBe(true)
  })

  it('échantillon suffisant -> non atténué', () => {
    const p = point({ part_isolee: couverture(12, 40, false) })
    expect(pointAttenue(p)).toBe(false)
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

// Lot C3 (D4) : le "gros point" par joueur — médiane de chaque axe (cohérent
// avec les lignes de repère du nuage, qui médianent déjà), taille au TOTAL
// des morts examinées.
describe('pointMedianJoueur', () => {
  it('rend null sans point', () => {
    expect(pointMedianJoueur([])).toBeNull()
  })

  it('médiane de chaque axe + somme des morts examinées (pas leur médiane)', () => {
    const pts = [
      point({ morts_examinees: 10, part_isolee: couverture(1, 10, true), couverture: couverture(1, 10, true) }),
      point({ morts_examinees: 20, part_isolee: couverture(5, 10, true), couverture: couverture(5, 10, true) }),
      point({ morts_examinees: 30, part_isolee: couverture(9, 10, true), couverture: couverture(9, 10, true) }),
    ]
    const med = pointMedianJoueur(pts)
    expect(med?.isolement).toBeCloseTo(0.5)
    expect(med?.couverture).toBeCloseTo(0.5)
    expect(med?.mortsExaminees).toBe(60)
  })

  it('un seul point : la médiane EST ce point, le total égale son décompte', () => {
    const p = point({ morts_examinees: 12, part_isolee: couverture(3, 10, true), couverture: couverture(7, 10, true) })
    const med = pointMedianJoueur([p])
    expect(med).toEqual({ isolement: 0.3, couverture: 0.7, mortsExaminees: 12 })
  })
})

describe('tailleMedianeDuPoint', () => {
  it('rend un point visiblement PLUS GROS que le plus gros point de session à décompte égal', () => {
    const total = 40
    expect(tailleMedianeDuPoint(total)).toBeGreaterThan(tailleDuPoint(total))
  })

  it('croît avec le total des morts examinées', () => {
    expect(tailleMedianeDuPoint(80)).toBeGreaterThan(tailleMedianeDuPoint(10))
  })

  it('plafonne pour un total très chargé', () => {
    expect(tailleMedianeDuPoint(10000)).toBe(tailleMedianeDuPoint(2000))
  })
})

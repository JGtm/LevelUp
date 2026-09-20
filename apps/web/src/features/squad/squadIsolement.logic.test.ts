import { describe, expect, it } from 'vitest'

import type { SquadIsolementMort, SquadIsolementRepere } from '@/lib/api/types'

import {
  contrasteIsolement,
  delaiSecondes,
  echelleAvecBande,
  echellesNuage,
  positionMort,
  positionRepere,
  repereAttenue,
  tailleRepere,
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

function mort(over: Partial<SquadIsolementMort> = {}): SquadIsolementMort {
  return {
    xuid: 'x1',
    gamertag: 'Alice',
    match_id: 'm1',
    time_ms: 10_000,
    distance_ratio: 0.6,
    hors_de_vue: false,
    vengee: true,
    delai_ms: 3_000,
    ...over,
  } as SquadIsolementMort
}

function repere(over: Partial<SquadIsolementRepere> = {}): SquadIsolementRepere {
  return {
    xuid: 'x1',
    gamertag: 'Alice',
    nb_morts: 10,
    mediane_distance_ratio: 0.8,
    mediane_delai_ms: 4_000,
    part_isolee: couverture(4, 10, true),
    couverture: couverture(6, 10, true),
    ...over,
  } as SquadIsolementRepere
}

describe('echelleAvecBande', () => {
  it('sans valeur, l’échelle tient sur son minimum et réserve quand même sa bande', () => {
    const e = echelleAvecBande([], 0.5, 2)
    expect(e.mesure).toBe(2)
    expect(e.bandeDebut).toBe(2.5)
    expect(e.max).toBe(3.5)
    expect(e.bandeCentre).toBe(3)
  })

  it('arrondit le haut de la zone mesurée au pas supérieur', () => {
    expect(echelleAvecBande([2.2], 0.5, 2).mesure).toBe(2.5)
    expect(echelleAvecBande([12.4], 1, 10).mesure).toBe(13)
  })

  it('la bande commence APRÈS la zone mesurée : aucune valeur réelle ne s’y pose', () => {
    const e = echelleAvecBande([3.1], 0.5, 2)
    expect(e.bandeDebut).toBeGreaterThan(e.mesure)
    expect(e.bandeCentre).toBeGreaterThan(e.bandeDebut)
    expect(e.max).toBeGreaterThanOrEqual(e.bandeCentre)
  })
})

describe('delaiSecondes', () => {
  it('rend le délai en secondes d’une mort vengée', () => {
    expect(delaiSecondes(mort({ delai_ms: 2_500 }))).toBe(2.5)
  })

  it('rend null pour une mort jamais vengée — jamais zéro', () => {
    expect(delaiSecondes(mort({ vengee: false, delai_ms: undefined }))).toBeNull()
  })
})

describe('positionMort', () => {
  const echelles = echellesNuage([mort({ distance_ratio: 1.2, delai_ms: 4_000 })])

  it('pose une mort mesurée à ses deux coordonnées réelles', () => {
    expect(positionMort(mort({ distance_ratio: 1.2, delai_ms: 4_000 }), echelles)).toEqual([1.2, 4])
  })

  it('pose une mort hors de vue dans la bande de distance', () => {
    const [x] = positionMort(mort({ distance_ratio: undefined, hors_de_vue: true }), echelles)
    expect(x).toBe(echelles.distance.bandeCentre)
  })

  it('pose une mort jamais vengée dans la bande de délai', () => {
    const [, y] = positionMort(mort({ vengee: false, delai_ms: undefined }), echelles)
    expect(y).toBe(echelles.delai.bandeCentre)
  })
})

describe('positionRepere', () => {
  const echelles = echellesNuage([mort()])

  it('pose le repère sur ses deux médianes', () => {
    expect(positionRepere(repere({ mediane_distance_ratio: 0.9, mediane_delai_ms: 5_000 }), echelles))
      .toEqual([0.9, 5])
  })

  it('renvoie le repère dans les bandes quand une médiane manque', () => {
    const [x, y] = positionRepere(
      repere({ mediane_distance_ratio: undefined, mediane_delai_ms: undefined }),
      echelles,
    )
    expect(x).toBe(echelles.distance.bandeCentre)
    expect(y).toBe(echelles.delai.bandeCentre)
  })
})

describe('tailleRepere', () => {
  it('projette le volume sur la plage réelle du roster', () => {
    expect(tailleRepere(10, 10, 50)).toBeLessThan(tailleRepere(50, 10, 50))
  })

  it('min === max : taille médiane de la plage, aucun écart à montrer', () => {
    expect(tailleRepere(30, 30, 30)).toBe(tailleRepere(99, 30, 30))
  })
})

describe('repereAttenue', () => {
  it('suit la réserve d’échantillon du taux d’isolement', () => {
    expect(repereAttenue(repere({ part_isolee: couverture(4, 10, true) }))).toBe(true)
    expect(repereAttenue(repere({ part_isolee: couverture(12, 40, false) }))).toBe(false)
  })
})

describe('contrasteIsolement', () => {
  it('rend null sous deux joueurs : une comparaison a besoin de deux termes', () => {
    expect(contrasteIsolement([])).toBeNull()
    expect(contrasteIsolement([repere()])).toBeNull()
  })

  it('désigne le plus exposé et le moins exposé', () => {
    const c = contrasteIsolement([
      repere({ gamertag: 'Alice', part_isolee: couverture(2, 10, false) }),
      repere({ gamertag: 'Bob', part_isolee: couverture(8, 10, false) }),
    ])
    expect(c?.loin.gamertag).toBe('Bob')
    expect(c?.proche.gamertag).toBe('Alice')
  })
})

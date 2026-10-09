/**
 * Tests de la logique pure de l'onglet Tendances : découpe par horizon (aux bornes), pas
 * proposés et par défaut, formatage par unité et par locale, borne des écarts.
 */
import { describe, expect, it } from 'vitest'

import type { TrendsPoint } from '@/lib/api/types'

import {
  HORIZONS,
  clampZ,
  defaultStep,
  formatTrendDelta,
  formatTrendValue,
  gameTypeOptions,
  pointsInHorizon,
  stepsForHorizon,
} from './tendances.logic'

const AS_OF = '2026-10-05T12:00:00Z'

function point(t: string): TrendsPoint {
  return { t, value: 1, matches: 1 }
}

describe('pointsInHorizon', () => {
  it('borne basse EXCLUE, borne haute INCLUSE : ]asOf - days, asOf]', () => {
    const points = [
      point('2026-09-28T12:00:00Z'), // exactement asOf - 7 j : exclu
      point('2026-09-28T12:00:01Z'), // juste après : inclus
      point('2026-10-05T12:00:00Z'), // exactement asOf : inclus
      point('2026-10-05T12:00:01Z'), // après asOf : exclu
    ]
    const gardes = pointsInHorizon(points, AS_OF, 7).map((p) => p.t)
    expect(gardes).toEqual(['2026-09-28T12:00:01Z', '2026-10-05T12:00:00Z'])
  })

  it('un point dont l’intervalle chevauche le début de l’horizon n’est pas gardé', () => {
    // début d’intervalle AVANT asOf - 30 j, même si l’intervalle recouvre l’horizon
    expect(pointsInHorizon([point('2026-09-04T00:00:00Z')], AS_OF, 30)).toEqual([])
  })

  it('accepte une série absente (null) et un instant Date', () => {
    expect(pointsInHorizon(null, AS_OF, 7)).toEqual([])
    expect(pointsInHorizon(undefined, new Date(AS_OF), 7)).toEqual([])
  })
})

describe('pas proposés et pas par défaut', () => {
  it('7 j : match, jour ; 30 j : match, jour, semaine ; 90 j : jour, semaine, mois ; 365 j : semaine, mois', () => {
    expect(stepsForHorizon(7)).toEqual(['match', 'day'])
    expect(stepsForHorizon(30)).toEqual(['match', 'day', 'week'])
    expect(stepsForHorizon(90)).toEqual(['day', 'week', 'month'])
    expect(stepsForHorizon(365)).toEqual(['week', 'month'])
  })

  it('défauts : match, jour, semaine, mois — et chaque défaut est un pas proposé', () => {
    expect(defaultStep(7)).toBe('match')
    expect(defaultStep(30)).toBe('day')
    expect(defaultStep(90)).toBe('week')
    expect(defaultStep(365)).toBe('month')
    for (const days of HORIZONS) {
      expect(stepsForHorizon(days)).toContain(defaultStep(days))
    }
  })
})

describe('clampZ', () => {
  it('borne à ± 2,5 et laisse passer l’intérieur', () => {
    expect(clampZ(7)).toBe(2.5)
    expect(clampZ(-7)).toBe(-2.5)
    expect(clampZ(2.5)).toBe(2.5)
    expect(clampZ(-1.25)).toBe(-1.25)
    expect(clampZ(0)).toBe(0)
  })
})

describe('formatTrendValue', () => {
  it('ratio : pourcentage, décimales de l’API moins 2 (3 → 1 décimale)', () => {
    expect(formatTrendValue(0.5432, 'ratio', 3, 'fr')).toBe('54,3\u00a0%')
    expect(formatTrendValue(0.5432, 'ratio', 3, 'en')).toBe('54.3%')
  })

  it('ratio : jamais moins de 0 décimale (2 → 0, 1 → 0)', () => {
    expect(formatTrendValue(0.5432, 'ratio', 2, 'fr')).toBe('54\u00a0%')
    expect(formatTrendValue(0.5432, 'ratio', 1, 'en')).toBe('54%')
  })

  it('secondes et heures : suffixe « s » et « h », séparateurs de la locale', () => {
    expect(formatTrendValue(1234.5, 'seconds', 1, 'en')).toBe('1,234.5\u00a0s')
    expect(formatTrendValue(12.25, 'hours', 1, 'fr')).toBe('12,3\u00a0h')
  })

  it('nombre : tel quel, avec le moins typographique pour un négatif', () => {
    expect(formatTrendValue(3.14159, 'number', 2, 'fr')).toBe('3,14')
    expect(formatTrendValue(-12.5, 'number', 1, 'en')).toBe('−12.5')
  })
})

describe('formatTrendValue — zéro sans signe', () => {
  it('−0,04 à une décimale s’écrit « 0,0 », jamais « −0,0 »', () => {
    expect(formatTrendValue(-0.04, 'number', 1, 'fr')).toBe('0,0')
    expect(formatTrendValue(-0.04, 'seconds', 1, 'en')).toBe('0.0 s')
    expect(formatTrendValue(-0.04, 'hours', 1, 'fr')).toBe('0,0 h')
  })

  it('ratio : −0,004 s’écrit « 0 % » au format de la locale', () => {
    expect(formatTrendValue(-0.004, 'ratio', 2, 'fr')).toBe('0 %')
    expect(formatTrendValue(-0.004, 'ratio', 2, 'en')).toBe('0%')
  })

  it('une vraie valeur négative garde son signe typographique', () => {
    expect(formatTrendValue(-0.5, 'number', 1, 'fr')).toBe('−0,5')
    expect(formatTrendValue(-0.05, 'ratio', 3, 'en')).toBe('−5.0%')
  })
})

describe('gameTypeOptions', () => {
  it('le type choisi est déjà dans la liste : liste inchangée', () => {
    expect(gameTypeOptions(['a', 'b'], 'b')).toEqual(['a', 'b'])
  })

  it('le type choisi est absent : ajouté en fin', () => {
    expect(gameTypeOptions(['a', 'b'], 'c')).toEqual(['a', 'b', 'c'])
    expect(gameTypeOptions([], 'c')).toEqual(['c'])
  })

  it('aucun type choisi : liste inchangée', () => {
    expect(gameTypeOptions(['a'], '')).toEqual(['a'])
  })
})

describe('formatTrendDelta', () => {
  it('ratio : points de pourcentage signés', () => {
    expect(formatTrendDelta(0.021, 'ratio', 3, 'fr')).toBe('+2,1\u00a0pts')
    expect(formatTrendDelta(-0.021, 'ratio', 3, 'en')).toBe('−2.1\u00a0pts')
  })

  it('autres unités : valeur signée, sans signe pour zéro', () => {
    expect(formatTrendDelta(1.5, 'number', 1, 'en')).toBe('+1.5')
    expect(formatTrendDelta(-3, 'seconds', 0, 'fr')).toBe('−3\u00a0s')
    expect(formatTrendDelta(0.001, 'number', 1, 'fr')).toBe('0,0')
  })
})

describe("arrondi d'Intl (pas toFixed)", () => {
  it("une valeur de mi-chemin s'écrit comme formatNumber", () => {
    expect(formatTrendValue(1.45, 'number', 1, 'fr')).toBe('1,5')
    expect(formatTrendValue(1.45, 'number', 1, 'en')).toBe('1.5')
    expect(formatTrendValue(1.005, 'number', 2, 'fr')).toBe('1,01')
    expect(formatTrendValue(2.675, 'number', 2, 'fr')).toBe('2,68')
  })

  it('écart : le signe suit le texte écrit', () => {
    expect(formatTrendDelta(0.145, 'number', 2, 'fr')).toBe('+0,15')
    expect(formatTrendDelta(-0.004, 'number', 2, 'fr')).toBe('0,00')
  })
})

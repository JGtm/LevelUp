import { describe, it, expect, vi } from 'vitest'

import {
  formatDateShort,
  formatNumber,
  outcomeColor,
  seriesColor,
  stackedAxisExtent,
  tickInterval,
} from './_utils'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

describe('_utils', () => {
  describe('outcomeColor', () => {
    it('mappe win → outcome-win', () => {
      expect(outcomeColor('win')).toBe('var(outcome-win)')
    })
    it('mappe loss → outcome-loss', () => {
      expect(outcomeColor('loss')).toBe('var(outcome-loss)')
    })
    it('mappe tie → outcome-draw', () => {
      expect(outcomeColor('tie')).toBe('var(outcome-draw)')
    })
    it('mappe dnf → outcome-dnf', () => {
      expect(outcomeColor('dnf')).toBe('var(outcome-dnf)')
    })
    it('fallback inconnu → chart-series-1', () => {
      expect(outcomeColor('unknown')).toBe('var(chart-series-1)')
      expect(outcomeColor(undefined)).toBe('var(chart-series-1)')
    })
  })

  describe('seriesColor', () => {
    it('cycle modulo 8', () => {
      expect(seriesColor(0)).toBe('var(chart-series-1)')
      expect(seriesColor(7)).toBe('var(chart-series-8)')
      expect(seriesColor(8)).toBe('var(chart-series-1)') // wrap
    })
  })

  describe('tickInterval', () => {
    it('petits volumes → 1', () => {
      expect(tickInterval(5)).toBe(1)
      expect(tickInterval(10)).toBe(1)
    })
    it('volumes moyens → 2', () => {
      expect(tickInterval(20)).toBe(2)
      expect(tickInterval(30)).toBe(2)
    })
    it('grand volume → 5', () => {
      expect(tickInterval(50)).toBe(5)
    })
    it('très grand → 10+', () => {
      expect(tickInterval(120)).toBe(10)
      expect(tickInterval(240)).toBeGreaterThan(10)
    })
  })

  describe('formatDateShort', () => {
    it('formate Date FR DD/MM', () => {
      expect(formatDateShort(new Date('2026-04-27'))).toBe('27/04')
    })
    it('accepte une string ISO', () => {
      expect(formatDateShort('2026-12-01')).toBe('01/12')
    })
  })

  describe('stackedAxisExtent — extent sur le JEU COMPLET, jamais sur les séries visibles', () => {
    it('aucune pile → {min: 0, max: 0}', () => {
      expect(stackedAxisExtent([])).toEqual({ min: 0, max: 0 })
      expect(stackedAxisExtent([], [])).toEqual({ min: 0, max: 0 })
    })

    it('une pile, une série : max = plus grande valeur, arrondi dizaine supérieure', () => {
      expect(stackedAxisExtent([[[5, 15, 8]]])).toEqual({ min: 0, max: 20 })
    })

    it('une pile, plusieurs séries EMPILÉES : max = plus grande SOMME par index (pas la plus grande valeur seule)', () => {
      // idx0 : 10+3=13 ; idx1 : 5+2=7. Max = 13 → dizaine sup. 20.
      expect(stackedAxisExtent([[[10, 5], [3, 2]]])).toEqual({ min: 0, max: 20 })
    })

    it('plusieurs piles indépendantes : retient le MAX entre piles, ne les additionne pas', () => {
      // Pile A (idx0=12), pile B (idx0=30) → max = 30, PAS 42.
      expect(stackedAxisExtent([[[12]], [[30]]])).toEqual({ min: 0, max: 30 })
    })

    it('negativeStacks omis → min toujours 0 (axe qui ne descend jamais sous zéro)', () => {
      expect(stackedAxisExtent([[[100]]])).toEqual({ min: 0, max: 100 })
    })

    it('negativeStacks : min = la somme la PLUS NÉGATIVE, arrondie à la dizaine inférieure (valeurs déjà signées)', () => {
      expect(stackedAxisExtent([], [[[-4, -6]]])).toEqual({ min: -10, max: 0 })
    })

    it('positif ET négatif ensemble (ex. butterfly kills/bonus vs morts)', () => {
      expect(stackedAxisExtent([[[12], [3]]], [[[-4]]])).toEqual({ min: -10, max: 20 })
    })

    it('null/undefined comptent pour 0 (comme ECharts)', () => {
      expect(stackedAxisExtent([[[null, 5, undefined]]])).toEqual({ min: 0, max: 10 })
    })

    it('valeur pile sur une dizaine → marge nulle (cas limite documenté, cohérent avec oneLifeWindowBoundsForData)', () => {
      expect(stackedAxisExtent([[[20]]])).toEqual({ min: 0, max: 20 })
    })

    it('indépendant de l\'ordre des piles/séries : le résultat ne dépend que des valeurs', () => {
      const a = stackedAxisExtent([[[30]], [[12]]], [[[-2]], [[-4]]])
      const b = stackedAxisExtent([[[12]], [[30]]], [[[-4]], [[-2]]])
      expect(a).toEqual(b)
    })
  })

  describe('formatNumber', () => {
    it('arrondit avec 1 décimale par défaut', () => {
      expect(formatNumber(3.456)).toBe('3.5')
    })
    it('respecte decimals', () => {
      expect(formatNumber(3.456, 2)).toBe('3.46')
    })
    it('NaN/Infinity → "-"', () => {
      expect(formatNumber(NaN)).toBe('-')
      expect(formatNumber(Infinity)).toBe('-')
    })
  })
})

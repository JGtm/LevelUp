import { describe, it, expect, vi } from 'vitest'

import {
  formatDateShort,
  formatNumber,
  outcomeColor,
  seriesColor,
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

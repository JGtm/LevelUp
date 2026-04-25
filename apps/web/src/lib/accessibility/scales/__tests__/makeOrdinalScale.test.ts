import { describe, it, expect } from 'vitest'
import { makeOrdinalScale } from '../makeOrdinalScale'

const scale = makeOrdinalScale({
  tiers: ['perf-tier-1', 'perf-tier-2', 'perf-tier-3'],
  thresholds: [80, 50],
})

describe('makeOrdinalScale', () => {
  describe('validation de config', () => {
    it('lève une erreur si tiers.length !== thresholds.length + 1', () => {
      expect(() => makeOrdinalScale({ tiers: ['perf-tier-1', 'perf-tier-2'], thresholds: [80, 50] }))
        .toThrow('tiers.length')
    })

    it('lève une erreur si thresholds non strictement décroissants', () => {
      expect(() => makeOrdinalScale({ tiers: ['perf-tier-1', 'perf-tier-2', 'perf-tier-3'], thresholds: [50, 80] }))
        .toThrow('décroissant')
    })
  })

  describe('mapping valeur → tier', () => {
    it('retourne tier[0] pour value >= threshold[0]', () => {
      expect(scale(80)).toBe('perf-tier-1')
      expect(scale(100)).toBe('perf-tier-1')
      expect(scale(80.1)).toBe('perf-tier-1')
    })

    it('retourne tier[0] exactement à la borne supérieure', () => {
      expect(scale(80)).toBe('perf-tier-1')
    })

    it('retourne tier[1] dans la tranche [50, 80[', () => {
      expect(scale(79.9)).toBe('perf-tier-2')
      expect(scale(65)).toBe('perf-tier-2')
      expect(scale(50)).toBe('perf-tier-2')
    })

    it('retourne tier[N-1] pour value < threshold[N-2]', () => {
      expect(scale(49.9)).toBe('perf-tier-3')
      expect(scale(0)).toBe('perf-tier-3')
      expect(scale(-1)).toBe('perf-tier-3')
    })
  })

  describe('valeurs limites', () => {
    it('NaN → tier le plus bas + warn (une seule fois)', () => {
      expect(scale(NaN)).toBe('perf-tier-3')
    })

    it('Infinity → tier le plus haut', () => {
      expect(scale(Infinity)).toBe('perf-tier-1')
    })

    it('-Infinity → tier le plus bas', () => {
      expect(scale(-Infinity)).toBe('perf-tier-3')
    })
  })

  describe('échelles concrètes (instances)', () => {
    it('perfScale — valeurs historiques connues', async () => {
      const { perfScale } = await import('../instances')
      expect(perfScale(85)).toBe('perf-tier-1') // Excellent
      expect(perfScale(70)).toBe('perf-tier-2') // Bon
      expect(perfScale(55)).toBe('perf-tier-3') // Correct
      expect(perfScale(40)).toBe('perf-tier-4') // Faible
      expect(perfScale(20)).toBe('perf-tier-5') // Mauvais
      expect(perfScale(80)).toBe('perf-tier-1') // Borne exacte
      expect(perfScale(35)).toBe('perf-tier-4') // Borne exacte
    })

    it('kdScale — décision §9.7 : ≥1 / [0,1[ / <0', async () => {
      const { kdScale } = await import('../instances')
      expect(kdScale(1.5)).toBe('perf-tier-1')
      expect(kdScale(1.0)).toBe('perf-tier-1') // borne exacte = bon
      expect(kdScale(0.9)).toBe('perf-tier-3')
      expect(kdScale(0.0)).toBe('perf-tier-3') // borne exacte = moyen
      expect(kdScale(-0.1)).toBe('perf-tier-5')
    })

    it('accuracyScale — seuils 55 / 40', async () => {
      const { accuracyScale } = await import('../instances')
      expect(accuracyScale(60)).toBe('perf-tier-1')
      expect(accuracyScale(45)).toBe('perf-tier-3')
      expect(accuracyScale(30)).toBe('perf-tier-5')
    })

    it('progressScale — 4 tiers 75 / 50 / 25', async () => {
      const { progressScale } = await import('../instances')
      expect(progressScale(80)).toBe('perf-tier-1')
      expect(progressScale(60)).toBe('perf-tier-2')
      expect(progressScale(30)).toBe('perf-tier-4')
      expect(progressScale(10)).toBe('perf-tier-5')
    })
  })
})

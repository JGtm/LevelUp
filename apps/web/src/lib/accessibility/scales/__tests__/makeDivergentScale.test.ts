import { describe, it, expect } from 'vitest'
import { makeDivergentScale } from '../makeDivergentScale'

const strictScale = makeDivergentScale({
  positive: 'divergent-pos',
  neutral: 'divergent-neutral',
  negative: 'divergent-neg',
  neutralBand: [0, 0],
})

const bandScale = makeDivergentScale({
  positive: 'divergent-pos',
  neutral: 'divergent-neutral',
  negative: 'divergent-neg',
  neutralBand: [-10, 10],
})

describe('makeDivergentScale', () => {
  describe('validation de config', () => {
    it('lève une erreur si neutralBand inversée', () => {
      expect(() => makeDivergentScale({
        positive: 'divergent-pos',
        neutral: 'divergent-neutral',
        negative: 'divergent-neg',
        neutralBand: [10, -10],
      })).toThrow('inversée')
    })
  })

  describe('mode strict [0, 0]', () => {
    it('> 0 → positif', () => expect(strictScale(0.1)).toBe('divergent-pos'))
    it('=== 0 → neutre', () => expect(strictScale(0)).toBe('divergent-neutral'))
    it('< 0 → négatif', () => expect(strictScale(-0.1)).toBe('divergent-neg'))
  })

  describe('mode bande [-10, 10]', () => {
    it('> 10 → positif', () => expect(bandScale(15)).toBe('divergent-pos'))
    it('=== 10 → neutre (borne inclusive)', () => expect(bandScale(10)).toBe('divergent-neutral'))
    it('entre -10 et 10 → neutre', () => expect(bandScale(5)).toBe('divergent-neutral'))
    it('=== -10 → neutre (borne inclusive)', () => expect(bandScale(-10)).toBe('divergent-neutral'))
    it('< -10 → négatif', () => expect(bandScale(-15)).toBe('divergent-neg'))
  })

  describe('valeurs limites', () => {
    it('NaN → neutre + warn', () => expect(strictScale(NaN)).toBe('divergent-neutral'))
    it('Infinity → positif', () => expect(strictScale(Infinity)).toBe('divergent-pos'))
    it('-Infinity → négatif', () => expect(strictScale(-Infinity)).toBe('divergent-neg'))
  })

  describe('instances concrètes', () => {
    it('mmrDeltaScale — bande ±10', async () => {
      const { mmrDeltaScale } = await import('../instances')
      expect(mmrDeltaScale(50)).toBe('divergent-pos')
      expect(mmrDeltaScale(5)).toBe('divergent-neutral')   // dans la bande
      expect(mmrDeltaScale(-5)).toBe('divergent-neutral')  // dans la bande
      expect(mmrDeltaScale(-50)).toBe('divergent-neg')
      expect(mmrDeltaScale(10)).toBe('divergent-neutral')  // borne inclusive
      expect(mmrDeltaScale(-10)).toBe('divergent-neutral') // borne inclusive
    })

    it('skillDeltaScale — strict zéro', async () => {
      const { skillDeltaScale } = await import('../instances')
      expect(skillDeltaScale(1)).toBe('divergent-pos')
      expect(skillDeltaScale(0)).toBe('divergent-neutral')
      expect(skillDeltaScale(-1)).toBe('divergent-neg')
    })
  })
})

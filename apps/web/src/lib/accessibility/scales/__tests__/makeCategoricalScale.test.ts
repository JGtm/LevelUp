import { describe, it, expect, beforeEach } from 'vitest'
import { makeCategoricalScale } from '../makeCategoricalScale'
import { log } from '../../_logger'

const scale = makeCategoricalScale({
  win:  'outcome-win',
  loss: 'outcome-loss',
  draw: 'outcome-draw',
})

beforeEach(() => log._resetForTests())

describe('makeCategoricalScale', () => {
  it('retourne le token pour une clé valide', () => {
    expect(scale('win')).toBe('outcome-win')
    expect(scale('loss')).toBe('outcome-loss')
    expect(scale('draw')).toBe('outcome-draw')
  })

  it('retourne null pour null/undefined', () => {
    expect(scale(null)).toBeNull()
    expect(scale(undefined)).toBeNull()
  })

  it('retourne null + loggue error pour clé inconnue', () => {
    expect(scale('unknown')).toBeNull()
  })

  describe('instances concrètes', () => {
    it('outcomeScale — toutes les clés connues', async () => {
      const { outcomeScale } = await import('../instances')
      expect(outcomeScale('win')).toBe('outcome-win')
      expect(outcomeScale('loss')).toBe('outcome-loss')
      expect(outcomeScale('draw')).toBe('outcome-draw')
      expect(outcomeScale('dnf')).toBe('outcome-dnf')
      expect(outcomeScale(null)).toBeNull()
    })

    it('narrativeScale — toutes les clés connues', async () => {
      const { narrativeScale } = await import('../instances')
      expect(narrativeScale('dominant')).toBe('narrative-dominant')
      expect(narrativeScale('humiliation')).toBe('narrative-humiliation')
      expect(narrativeScale('remontada')).toBe('narrative-remontada')
      expect(narrativeScale('debacle')).toBe('narrative-debacle')
      expect(narrativeScale('contre_remontada')).toBe('narrative-contre-remontada')
      expect(narrativeScale(null)).toBeNull()
    })
  })
})

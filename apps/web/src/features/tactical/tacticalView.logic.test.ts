/**
 * tacticalView.logic.test.ts — Tests unitaires pour la logique tactique.
 */
import { describe, it, expect } from 'vitest'
import {
  generateAnalysisTitle,
  generateProcessingMessage,
  getQuestionLabel,
  getQuestionUnit,
  isDivergentQuestion,
  isHeatQuestion,
  isRoutesQuestion,
  validateQuestion,
  validateWho,
  type QuestionType,
  type WhoType,
} from './tacticalView.logic'

describe('tacticalView.logic', () => {
  describe('generateAnalysisTitle', () => {
    it('should generate FR title correctly', () => {
      const title = generateAnalysisTitle('Streets', 'morts', 'fr')
      expect(title).toBe('Plan de Streets — Où je meurs')
    })

    it('should generate EN title correctly', () => {
      const title = generateAnalysisTitle('Streets', 'morts', 'en')
      expect(title).toBe('Streets Plan — Where I Die')
    })

    it('should handle different questions', () => {
      const titleKills = generateAnalysisTitle('Aquarius', 'kills', 'fr')
      expect(titleKills).toBe('Plan de Aquarius — Où je tue')

      const titleTime = generateAnalysisTitle('Bazaar', 'temps', 'fr')
      expect(titleTime).toBe('Plan de Bazaar — Où je passe mon temps')
    })
  })

  describe('generateProcessingMessage', () => {
    it('should return processing status when matches are pending', () => {
      const result = generateProcessingMessage(5, 0, 'fr')
      expect(result.status).toBe('processing')
      expect(result.message).toBe('Traitement en cours : 5 matchs')
    })

    it('should return unavailable status when matches are not cookable', () => {
      const result = generateProcessingMessage(0, 3, 'fr')
      expect(result.status).toBe('unavailable')
      expect(result.message).toBe('Données non disponibles pour 3 matchs')
    })

    it('should return idle status when no pending or unavailable matches', () => {
      const result = generateProcessingMessage(0, 0, 'fr')
      expect(result.status).toBe('idle')
      expect(result.message).toBe('')
    })

    it('should handle EN locale', () => {
      const result = generateProcessingMessage(2, 0, 'en')
      expect(result.message).toBe('Processing: 2 matches')
    })

    it('should prioritize pending over unavailable', () => {
      const result = generateProcessingMessage(5, 3, 'fr')
      expect(result.status).toBe('processing')
      expect(result.message).toContain('5')
    })
  })

  describe('getQuestionLabel', () => {
    it('should return FR label', () => {
      expect(getQuestionLabel('morts', 'fr')).toBe('Où je meurs')
      expect(getQuestionLabel('kills', 'fr')).toBe('Où je tue')
    })

    it('should return EN label', () => {
      expect(getQuestionLabel('morts', 'en')).toBe('Where I Die')
      expect(getQuestionLabel('kills', 'en')).toBe('Where I Kill')
    })
  })

  describe('getQuestionUnit', () => {
    it('should return correct FR unit', () => {
      expect(getQuestionUnit('morts', 'fr')).toBe('morts par match')
      expect(getQuestionUnit('temps', 'fr')).toBe('s par match')
      expect(getQuestionUnit('gagne', 'fr')).toBe('écart V moins D')
    })

    it('should return correct EN unit', () => {
      expect(getQuestionUnit('morts', 'en')).toBe('deaths per match')
      expect(getQuestionUnit('temps', 'en')).toBe('s per match')
    })

    it('should return empty string for routes', () => {
      expect(getQuestionUnit('routes', 'fr')).toBe('')
      expect(getQuestionUnit('routes', 'en')).toBe('')
    })
  })

  describe('validateQuestion', () => {
    it('should accept valid questions', () => {
      const validQuestions: QuestionType[] = ['morts', 'kills', 'gagne', 'temps', 'routes', 'isole']
      validQuestions.forEach((q) => {
        expect(validateQuestion(q)).toBe(true)
      })
    })

    it('should reject invalid questions', () => {
      expect(validateQuestion('invalid')).toBe(false)
      expect(validateQuestion('frag')).toBe(false)
    })
  })

  describe('validateWho', () => {
    it('should accept valid who values', () => {
      const validWhos: WhoType[] = ['moi', 'escouade', 'adv']
      validWhos.forEach((w) => {
        expect(validateWho(w)).toBe(true)
      })
    })

    it('should reject invalid who values', () => {
      expect(validateWho('everyone')).toBe(false)
      expect(validateWho('squad')).toBe(false)
    })
  })

  describe('isRoutesQuestion', () => {
    it('should return true for routes', () => {
      expect(isRoutesQuestion('routes')).toBe(true)
    })

    it('should return false for other questions', () => {
      expect(isRoutesQuestion('morts')).toBe(false)
      expect(isRoutesQuestion('kills')).toBe(false)
    })
  })

  describe('isHeatQuestion', () => {
    it('should return true for heat questions', () => {
      expect(isHeatQuestion('morts')).toBe(true)
      expect(isHeatQuestion('kills')).toBe(true)
      expect(isHeatQuestion('temps')).toBe(true)
    })

    it('should return false for non-heat questions', () => {
      expect(isHeatQuestion('routes')).toBe(false)
    })
  })

  describe('isDivergentQuestion', () => {
    it('should return true for divergent questions', () => {
      expect(isDivergentQuestion('gagne')).toBe(true)
    })

    it('should return false for non-divergent questions', () => {
      expect(isDivergentQuestion('morts')).toBe(false)
      expect(isDivergentQuestion('kills')).toBe(false)
      expect(isDivergentQuestion('routes')).toBe(false)
    })
  })
})

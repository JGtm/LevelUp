import { describe, expect, it } from 'vitest'
import { getMatchCardOutcomeStyle, getMatchNarrativeBadgeMeta } from './match-card-presentation'

describe('match-card-presentation', () => {
  it('réutilise la palette legacy pour les scores victoire et défaite', () => {
    expect(getMatchCardOutcomeStyle('win').scoreColor).toBe('#4CAF50')
    expect(getMatchCardOutcomeStyle('loss').scoreColor).toBe('#F44336')
  })

  it('mappe les badges narratifs supportés', () => {
    expect(getMatchNarrativeBadgeMeta('dominant')).toEqual({
      label: 'DOMINATION',
      color: '#00DC82',
      textColor: '#052e16',
    })
    expect(getMatchNarrativeBadgeMeta('remontada')).toEqual({
      label: 'REMONTADA',
      color: '#0072B2',
      textColor: '#f8fafc',
    })
    expect(getMatchNarrativeBadgeMeta('unknown')).toBeNull()
  })
})

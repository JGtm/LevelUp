import { describe, expect, it } from 'vitest'
import { getMatchCardOutcomeStyle, getMatchNarrativeBadgeMeta } from './match-card-presentation'

describe('match-card-presentation', () => {
  it('retourne des CSS vars de palette pour les scores victoire et défaite', () => {
    expect(getMatchCardOutcomeStyle('win').scoreColor).toBe('var(--ac-outcome-win)')
    expect(getMatchCardOutcomeStyle('loss').scoreColor).toBe('var(--ac-outcome-loss)')
  })

  it('mappe les badges narratifs supportés avec des CSS vars de palette', () => {
    expect(getMatchNarrativeBadgeMeta('dominant')).toEqual({
      label: 'DOMINATION',
      color: 'var(--ac-narrative-dominant)',
      textColor: 'var(--ac-narrative-dominant-text)',
    })
    expect(getMatchNarrativeBadgeMeta('remontada')).toEqual({
      label: 'REMONTADA',
      color: 'var(--ac-narrative-remontada)',
      textColor: 'var(--ac-narrative-remontada-text)',
    })
    expect(getMatchNarrativeBadgeMeta('unknown')).toBeNull()
  })
})

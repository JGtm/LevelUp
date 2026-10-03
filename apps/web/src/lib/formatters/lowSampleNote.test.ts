import { describe, it, expect } from 'vitest'
import { withLowSampleNote } from './lowSampleNote'

describe('withLowSampleNote', () => {
  it('sans réserve, rend le texte tel quel', () => {
    expect(withLowSampleNote('12 / 40', false, 'échantillon faible')).toBe('12 / 40')
  })

  it('avec réserve, accole la note après le séparateur texte par défaut', () => {
    expect(withLowSampleNote('12 / 40', true, 'échantillon faible')).toBe(
      '12 / 40 — échantillon faible',
    )
  })

  it('accepte un séparateur HTML pour un tooltip', () => {
    expect(withLowSampleNote('ligne', true, 'low sample', '<br/>')).toBe('ligne<br/>low sample')
  })

  it('ne cache jamais la valeur : la note est un suffixe, pas un remplacement', () => {
    const out = withLowSampleNote('valeur', true, 'note')
    expect(out.startsWith('valeur')).toBe(true)
  })
})

import { describe, expect, it } from 'vitest'
import { fillTemplate, substitute } from './template'

describe('fillTemplate', () => {
  it('remplace les {nom} par leurs valeurs, nombres compris', () => {
    expect(fillTemplate('{a} et {b}', { a: 1, b: 'deux' })).toBe('1 et deux')
  })

  it('laisse tel quel un nom absent, null ou undefined', () => {
    expect(fillTemplate('{a} {x} {y} {z}', { a: 1, y: null, z: undefined })).toBe('1 {x} {y} {z}')
  })

  it('rend le gabarit sans paramètres', () => {
    expect(fillTemplate('Bonjour {nom}')).toBe('Bonjour {nom}')
  })

  it('garde un zéro et la chaîne vide : seules les valeurs absentes laissent le motif', () => {
    expect(fillTemplate('[{n}] [{s}]', { n: 0, s: '' })).toBe('[0] []')
  })
})

describe('substitute', () => {
  it('applique la règle propre du gabarit ; undefined et null laissent le motif', () => {
    const resolve = (name: string) => (name === 'plural' ? 's' : name === 'vide' ? '' : name === 'nul' ? null : undefined)
    expect(substitute('jour{plural} {vide}|{nul}|{autre}', resolve)).toBe('jours |{nul}|{autre}')
  })
})

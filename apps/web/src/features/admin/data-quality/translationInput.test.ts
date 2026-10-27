import { describe, expect, it } from 'vitest'

import { KNOWN_LOCALES } from '@/lib/i18n/locale'
import { namesFromValues, translationFields } from './translationInput'

describe('translationFields', () => {
  it('produit un champ par langue de la liste, dans son ordre', () => {
    const fields = translationFields((l) => `Nom ${l}`, 'cle')
    expect(fields.map((f) => f.key)).toEqual([...KNOWN_LOCALES])
    expect(fields.every((f) => f.initial === '' && f.placeholder === 'cle')).toBe(true)
    expect(fields[0].label).toBe(`Nom ${KNOWN_LOCALES[0]}`)
  })
})

describe('namesFromValues', () => {
  it('rogne les noms, écarte les vides et les clés hors liste', () => {
    const values: Record<string, string> = { zz: 'inconnue', [KNOWN_LOCALES[0]]: '  Assassin ' }
    for (const l of KNOWN_LOCALES.slice(1)) values[l] = '   '
    expect(namesFromValues(values)).toEqual({ [KNOWN_LOCALES[0]]: 'Assassin' })
  })

  it('retourne une carte vide quand rien n\'est saisi', () => {
    expect(namesFromValues({})).toEqual({})
  })
})

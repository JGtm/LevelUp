import { describe, it, expect } from 'vitest'

import { DEFAULT_EFFECTIVE_HP_TO_KILL, getHelpText } from './i18n'

/** Concatène tout le texte combat d'un glossaire pour des assertions globales. */
function flatten(locale: 'fr' | 'en', hp?: number): string {
  const text = hp === undefined ? getHelpText(locale) : getHelpText(locale, hp)
  return text.glossary.sections
    .flatMap((s) => s.entries)
    .flatMap((e) => [e.term, e.definition, e.formula ?? '', e.example ?? ''])
    .join('\n')
}

describe('getHelpText — copy combat title-aware', () => {
  it('ne laisse jamais fuiter le jeton {{HP}} (fr/en, défaut et 115)', () => {
    expect(flatten('fr')).not.toContain('{{HP}}')
    expect(flatten('en')).not.toContain('{{HP}}')
    expect(flatten('fr', 115)).not.toContain('{{HP}}')
    expect(flatten('en', 115)).not.toContain('{{HP}}')
  })

  it('par défaut, applique le barème Halo Infinite (225)', () => {
    expect(DEFAULT_EFFECTIVE_HP_TO_KILL).toBe(225)
    const fr = flatten('fr')
    expect(fr).toContain('225 × (éliminations + assistances/3) / dégâts infligés')
    expect(fr).toContain('dégâts reçus / (225 × morts)')
  })

  it('injecte le barème du titre courant dans le rendement et la résistance (115)', () => {
    const fr = flatten('fr', 115)
    expect(fr).toContain('115 × (éliminations + assistances/3) / dégâts infligés')
    expect(fr).toContain('dégâts reçus / (115 × morts)')
    // Tableau LUSR : la cellule composite suit le même barème.
    expect(fr).toContain('115 × (frags + ass/3) / dégâts')

    const en = flatten('en', 115)
    expect(en).toContain('Offensive conversion = 115 × (kills + assists/3) / damage_dealt')
    expect(en).toContain('Defensive resistance = damage_taken / (115 × deaths)')
  })

  it('conserve les seuils P80 calibrés Halo Infinite quel que soit le barème', () => {
    // Les P80 (0,83 / 1,59) ne sont pas tokenisés : recalibration par titre différée.
    expect(flatten('fr', 115)).toContain('0,83')
    expect(flatten('fr', 115)).toContain('1,59')
  })
})

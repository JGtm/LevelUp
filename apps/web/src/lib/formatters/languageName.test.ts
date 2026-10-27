import { describe, expect, it } from 'vitest'

import { KNOWN_LOCALES } from '@/lib/i18n/locale'
import { languageName } from './languageName'

describe('languageName', () => {
  it('nomme chaque langue de la liste dans la langue de l\'interface', () => {
    for (const ui of KNOWN_LOCALES) {
      for (const code of KNOWN_LOCALES) {
        const name = languageName(code, ui)
        expect(name.length).toBeGreaterThan(0)
      }
    }
  })

  it('rend un nom différent selon la langue de l\'interface', () => {
    expect(languageName('en', 'fr')).not.toBe(languageName('en', 'en'))
  })
})

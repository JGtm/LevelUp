import { describe, expect, it } from 'vitest'
import { resolveBootLocale } from './bootLocale'
import { DEFAULT_LOCALE } from './locale'

describe('resolveBootLocale', () => {
  it('le segment de langue de l’URL prime sur tout', () => {
    expect(resolveBootLocale('/en/t/halo_infinite/explorer', 'fr', ['fr-FR'])).toBe('en')
  })

  it('sans segment, la dernière locale de session prime sur le navigateur', () => {
    expect(resolveBootLocale('/', 'fr', ['en-US'])).toBe('fr')
  })

  it('une locale mémorisée inconnue est ignorée', () => {
    expect(resolveBootLocale('/', 'de', ['en-GB'])).toBe('en')
  })

  it('sans mémoire, première langue connue du navigateur', () => {
    expect(resolveBootLocale('/', null, ['de-DE', 'EN-us', 'fr'])).toBe('en')
  })

  it('rien de connu → DEFAULT_LOCALE', () => {
    expect(resolveBootLocale('/', null, ['de-DE'])).toBe(DEFAULT_LOCALE)
    expect(resolveBootLocale('/', null, [])).toBe(DEFAULT_LOCALE)
  })
})

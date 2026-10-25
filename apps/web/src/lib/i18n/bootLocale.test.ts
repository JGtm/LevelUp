import { describe, expect, it } from 'vitest'
import { resolveBootLocale } from './bootLocale'

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

  it('rien de connu → français', () => {
    expect(resolveBootLocale('/', null, ['de-DE'])).toBe('fr')
    expect(resolveBootLocale('/', null, [])).toBe('fr')
  })
})

/**
 * Tests — parseRouteSegments (module title-routing, D-10/D-11).
 *
 * Fonction PURE : extrait langue + slug de titre d'un pathname sous le namespace
 * `/t/`. Aucune validation de la VALEUR du slug (D-2, verbatim).
 */
import { describe, it, expect } from 'vitest'
import { parseRouteSegments } from './parseRouteSegments'

describe('parseRouteSegments', () => {
  it('capture le titre sous /t/{slug} (sans langue)', () => {
    expect(parseRouteSegments('/t/halo_infinite/players/x/home')).toEqual({
      titleSlug: 'halo_infinite',
    })
  })

  it('capture langue + titre sous /{lang}/t/{slug}', () => {
    expect(parseRouteSegments('/en/t/halo_5/players/x/stats/timeseries')).toEqual({
      lang: 'en',
      titleSlug: 'halo_5',
    })
  })

  it('langue fr reconnue', () => {
    expect(parseRouteSegments('/fr/t/halo_infinite/players/x')).toEqual({
      lang: 'fr',
      titleSlug: 'halo_infinite',
    })
  })

  it('page joueur legacy sans segment → rien', () => {
    expect(parseRouteSegments('/players/x/home')).toEqual({})
  })

  it('page agnostique → rien', () => {
    expect(parseRouteSegments('/settings')).toEqual({})
  })

  it('langue sur page agnostique NON capturée (D-3 : /en/settings)', () => {
    expect(parseRouteSegments('/en/settings')).toEqual({})
  })

  it('/t seul → rien', () => {
    expect(parseRouteSegments('/t')).toEqual({})
  })

  it('/t/ (trailing slash, pas de slug) → rien', () => {
    expect(parseRouteSegments('/t/')).toEqual({})
  })

  it('/en/t (langue + t sans slug) → rien', () => {
    expect(parseRouteSegments('/en/t')).toEqual({})
  })

  it('racine / → rien', () => {
    expect(parseRouteSegments('/')).toEqual({})
  })

  it('chaîne vide → rien', () => {
    expect(parseRouteSegments('')).toEqual({})
  })

  it('/t/x/ (trailing slash) → titleSlug=x', () => {
    expect(parseRouteSegments('/t/x/')).toEqual({ titleSlug: 'x' })
  })

  it('double slash toléré (segments vides ignorés)', () => {
    expect(parseRouteSegments('/t//halo_5//players')).toEqual({ titleSlug: 'halo_5' })
    expect(parseRouteSegments('//t/halo_5')).toEqual({ titleSlug: 'halo_5' })
  })

  it('slug pris VERBATIM (aucune validation de valeur — D-2)', () => {
    expect(parseRouteSegments('/t/inconnu_xyz/players/x')).toEqual({ titleSlug: 'inconnu_xyz' })
  })

  it('une langue inconnue en tête n’est pas capturée et ne libère pas le titre', () => {
    // 'de' ∉ locales connues → pas de lang ; et segs[0] !== 't' → pas de titre.
    expect(parseRouteSegments('/de/t/halo_5/players/x')).toEqual({})
  })
})

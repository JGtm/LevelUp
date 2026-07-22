import { describe, it, expect } from 'vitest'

import { playerRelativePath, routeTemplateSuffix, playerScopedHref } from './playerScopedPath'

describe('playerRelativePath', () => {
  it('extrait le suffixe (forme courte /t/)', () => {
    expect(playerRelativePath('/t/halo_infinite/players/JGtm/stats/sessions')).toBe(
      '/stats/sessions',
    )
  })

  it('extrait le suffixe avec segment langue', () => {
    expect(playerRelativePath('/en/t/halo_5/players/x/home')).toBe('/home')
  })

  it('tolère les anciens pathnames /players/…', () => {
    expect(playerRelativePath('/players/x/community/relations')).toBe('/community/relations')
  })

  it('racine joueur nue → chaîne vide', () => {
    expect(playerRelativePath('/t/halo_infinite/players/x')).toBe('')
  })

  it('page agnostique → null', () => {
    expect(playerRelativePath('/settings')).toBeNull()
    expect(playerRelativePath('/')).toBeNull()
    expect(playerRelativePath('/admin/sync')).toBeNull()
  })
})

describe('routeTemplateSuffix', () => {
  it('retourne le suffixe après players/$playerSlug', () => {
    expect(
      routeTemplateSuffix('/{-$lang}/t/$titleSlug/players/$playerSlug/career/citations'),
    ).toBe('/career/citations')
  })

  it('racine joueur → chaîne vide', () => {
    expect(routeTemplateSuffix('/{-$lang}/t/$titleSlug/players/$playerSlug')).toBe('')
  })

  it('template hors scope joueur → chaîne vide', () => {
    expect(routeTemplateSuffix('/settings')).toBe('')
  })
})

describe('playerScopedHref', () => {
  it('construit un href title-scoped avec suffixe', () => {
    expect(playerScopedHref('halo_infinite', 'JGtm', '/matches/m-42')).toBe(
      '/t/halo_infinite/players/JGtm/matches/m-42',
    )
  })

  it('encode le playerSlug (dérivé utilisateur)', () => {
    expect(playerScopedHref('halo_5', 'jg tm', '/explorer')).toBe(
      '/t/halo_5/players/jg%20tm/explorer',
    )
  })

  it('préserve un suffixe portant query string', () => {
    expect(
      playerScopedHref('halo_infinite', 'p1', '/explorer?mode=player&target=Bob'),
    ).toBe('/t/halo_infinite/players/p1/explorer?mode=player&target=Bob')
  })

  it('suffixe par défaut vide → racine joueur', () => {
    expect(playerScopedHref('halo_infinite', 'p1')).toBe('/t/halo_infinite/players/p1')
  })
})

import { describe, it, expect } from 'vitest'

import { resolvePageTitle } from './pageTitle'

describe('resolvePageTitle (URLs title-scoped)', () => {
  it('override par suffixe joueur (forme courte /t/)', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/jgtm/stats/timeseries')).toBe(
      'LevelUp - Séries temporelles',
    )
  })

  it('override par suffixe joueur avec segment langue', () => {
    expect(resolvePageTitle('/en/t/halo_5/players/x/career/citations')).toBe('LevelUp - Citations')
  })

  it('titre dérivé d’un item de nav (accueil)', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/x/home')).toBe('LevelUp - Accueil')
  })

  it('page de match', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/x/matches/abc-123')).toBe('LevelUp - Match')
  })

  it('racine joueur nue → Accueil', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/x')).toBe('LevelUp - Accueil')
  })

  it('page statique (agnostique)', () => {
    expect(resolvePageTitle('/settings')).toBe('LevelUp - Parametres')
    expect(resolvePageTitle('/')).toBe('LevelUp - Accueil')
  })

  it('suffixe inconnu → LevelUp', () => {
    expect(resolvePageTitle('/t/halo_infinite/players/x/zzz-inconnu')).toBe('LevelUp')
  })
})

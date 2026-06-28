import { describe, expect, it } from 'vitest'

import { buildPlayerDestination, PLAYER_PRIMARY_NAV_ITEMS } from './shellNavigation'

describe('buildPlayerDestination', () => {
  it('redirige vers home si aucun slug courant', () => {
    expect(buildPlayerDestination('/', null, 'new-player')).toBe('/players/new-player/home')
  })

  it('préserve la section active quand le path est dans le scope joueur', () => {
    expect(
      buildPlayerDestination(
        '/players/old-player/stats/timeseries',
        'old-player',
        'new-player',
      ),
    ).toBe('/players/new-player/stats/timeseries')
  })

  it('retombe sur home si le path actuel n’est pas un path joueur', () => {
    expect(buildPlayerDestination('/settings', 'old-player', 'new-player')).toBe(
      '/players/new-player/home',
    )
  })

  it('retombe sur home si le path est la racine du joueur', () => {
    expect(buildPlayerDestination('/players/old-player', 'old-player', 'new-player')).toBe(
      '/players/new-player/home',
    )
  })

  it('place Communauté avant Carrière dans le parcours principal', () => {
    const labels = PLAYER_PRIMARY_NAV_ITEMS.map((item) => item.label)

    expect(labels.indexOf('Communauté')).toBeGreaterThanOrEqual(0)
    expect(labels.indexOf('Communauté')).toBeLessThan(labels.indexOf('Carrière'))
  })
})

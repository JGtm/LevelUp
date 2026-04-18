import { describe, expect, it } from 'vitest'

import { buildPlayerDestination } from './shellNavigation'

describe('buildPlayerDestination', () => {
  it('redirige vers home si aucun slug courant', () => {
    expect(buildPlayerDestination('/', null, 'new-player')).toBe('/players/new-player/home')
  })

  it('préserve la section active quand le path est dans le scope joueur', () => {
    expect(
      buildPlayerDestination(
        '/players/old-player/stats/history',
        'old-player',
        'new-player',
      ),
    ).toBe('/players/new-player/stats/history')
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
})

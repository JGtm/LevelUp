import { describe, expect, it } from 'vitest'

import {
  buildPlayerDestination,
  PLAYER_PRIMARY_NAV_ITEMS,
  resolveIndexRedirect,
  type IndexRedirectInput,
} from './shellNavigation'

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

describe('resolveIndexRedirect', () => {
  const base: IndexRedirectInput = {
    isBootstrapped: true,
    authMode: 'none',
    currentUsername: null,
    currentPlayerSlug: null,
    firstAvailablePlayerSlug: null,
  }

  it('attend l’hydratation avant toute décision', () => {
    // authMode par défaut 'none' pré-hydratation ne doit pas être interprété.
    expect(resolveIndexRedirect({ ...base, isBootstrapped: false, authMode: 'xbox' })).toEqual({
      kind: 'wait',
    })
  })

  it('anonyme en mode xbox → /login (régression déconnexion : jamais de fallback sans shell)', () => {
    expect(resolveIndexRedirect({ ...base, authMode: 'xbox', currentUsername: null })).toEqual({
      kind: 'login',
    })
  })

  it('anonyme en mode password → /login', () => {
    expect(resolveIndexRedirect({ ...base, authMode: 'password', currentUsername: null })).toEqual({
      kind: 'login',
    })
  })

  it('anonyme en mode xbox avec des joueurs listés → /login (priorité sur player)', () => {
    // Filet : même si un jour available_players fuyait en anonyme, on ne rend jamais
    // une home joueur sans shell — on renvoie vers /login.
    expect(
      resolveIndexRedirect({
        ...base,
        authMode: 'xbox',
        currentUsername: null,
        firstAvailablePlayerSlug: 'someone',
      }),
    ).toEqual({ kind: 'login' })
  })

  it('connecté avec un joueur courant → sa home', () => {
    expect(
      resolveIndexRedirect({
        ...base,
        authMode: 'xbox',
        currentUsername: 'alice',
        currentPlayerSlug: 'alice',
      }),
    ).toEqual({ kind: 'player', slug: 'alice' })
  })

  it('connecté sans joueur courant mais avec un disponible → premier disponible', () => {
    expect(
      resolveIndexRedirect({
        ...base,
        authMode: 'password',
        currentUsername: 'alice',
        firstAvailablePlayerSlug: 'first',
      }),
    ).toEqual({ kind: 'player', slug: 'first' })
  })

  it('mode public/démo (none) avec un joueur → home joueur, jamais /login', () => {
    expect(
      resolveIndexRedirect({ ...base, authMode: 'none', firstAvailablePlayerSlug: 'pub' }),
    ).toEqual({ kind: 'player', slug: 'pub' })
  })

  it('hydraté sans aucun joueur → setup', () => {
    expect(resolveIndexRedirect({ ...base, authMode: 'none' })).toEqual({ kind: 'setup' })
  })
})

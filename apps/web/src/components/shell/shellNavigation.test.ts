import { describe, expect, it } from 'vitest'

import {
  resolveIndexRedirect,
  resolvePlayerFallback,
  resolvePlayerSwitch,
  type IndexRedirectInput,
} from './shellNavigation'

describe('resolvePlayerSwitch', () => {
  it('reste sur la même route (préserve la sous-page) quand on est sous une sous-page joueur', () => {
    expect(resolvePlayerSwitch('/t/halo_infinite/players/old/stats/timeseries')).toEqual({
      kind: 'same-route',
    })
    // Tolère aussi les anciens pathnames /players/… (préfixe titre optionnel).
    expect(resolvePlayerSwitch('/players/old/community/relations')).toEqual({ kind: 'same-route' })
  })

  it('retombe sur home si le path n’est pas une sous-page joueur', () => {
    expect(resolvePlayerSwitch('/settings')).toEqual({ kind: 'home' })
    expect(resolvePlayerSwitch('/')).toEqual({ kind: 'home' })
  })

  it('retombe sur home si le path est la racine joueur nue', () => {
    expect(resolvePlayerSwitch('/t/halo_infinite/players/old')).toEqual({ kind: 'home' })
  })
})

describe('resolvePlayerFallback', () => {
  const players = [{ player_slug: 'alice' }, { player_slug: 'bob' }]

  it('slug d’URL présent → ok (pas de redirection)', () => {
    expect(resolvePlayerFallback('bob', players)).toEqual({ kind: 'ok' })
  })

  it('slug d’URL inconnu → premier joueur disponible', () => {
    expect(resolvePlayerFallback('ghost', players)).toEqual({ kind: 'redirect', slug: 'alice' })
  })

  it('aucun joueur disponible → index', () => {
    expect(resolvePlayerFallback('anyone', [])).toEqual({ kind: 'index' })
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

import { describe, expect, it } from 'vitest'
import type { HomeMatchesPage, RecentMatchItem } from '@/lib/api/types'
import {
  HOME_TILES_PAGE_SIZE,
  homeMatchesUrl,
  mergeHomeTiles,
  patchFavoriteFlag,
  type HomeTilesHead,
} from './homeTiles.logic'

const tile = (id: string, fav = false): RecentMatchItem => ({ match_id: id, is_favorite: fav }) as RecentMatchItem
const tiles = (...ids: string[]) => ids.map((id) => tile(id))

const head = (ids: string[], more: boolean, next?: string): HomeTilesHead => ({
  recent_matches: tiles(...ids),
  recent_matches_has_more: more,
  recent_matches_next_before: next,
})

describe('mergeHomeTiles', () => {
  it('sans page : la tête seule, la suite annoncée par la tête', () => {
    const r = mergeHomeTiles(head(['a', 'b'], true, 'c0'), [])
    expect(r.tiles.map((t) => t.match_id)).toEqual(['a', 'b'])
    expect(r.hasMore).toBe(true)
    expect(r.nextBefore).toBe('c0')
  })

  it('tête + pages chargées dans l\'ordre ; la suite vient de la dernière page chargée', () => {
    const p1: HomeMatchesPage = { items: tiles('c', 'd'), has_more: true, next_before: 'c1' }
    const p2: HomeMatchesPage = { items: tiles('e'), has_more: false }
    const r = mergeHomeTiles(head(['a', 'b'], true, 'c0'), [p1, p2])
    expect(r.tiles.map((t) => t.match_id)).toEqual(['a', 'b', 'c', 'd', 'e'])
    expect(r.hasMore).toBe(false)
    expect(r.nextBefore).toBeUndefined()
  })

  it('une page pas encore chargée arrête l\'assemblage (jamais de tuile au-delà d\'un trou)', () => {
    const p1: HomeMatchesPage = { items: tiles('c'), has_more: true, next_before: 'c1' }
    const p3: HomeMatchesPage = { items: tiles('z'), has_more: false }
    const r = mergeHomeTiles(head(['a'], true, 'c0'), [p1, undefined, p3])
    expect(r.tiles.map((t) => t.match_id)).toEqual(['a', 'c'])
    expect(r.nextBefore).toBe('c1')
  })

  it('une tuile déjà vue n\'est jamais affichée deux fois', () => {
    const p1: HomeMatchesPage = { items: tiles('b', 'c'), has_more: false }
    expect(mergeHomeTiles(head(['a', 'b'], true, 'c0'), [p1]).tiles.map((t) => t.match_id)).toEqual(['a', 'b', 'c'])
  })

  it('has_more sans curseur : pas de suite demandable', () => {
    const r = mergeHomeTiles(head(['a'], true, undefined), [])
    expect(r.hasMore).toBe(false)
  })

  it('tête sans liste (champ absent) : liste vide', () => {
    const r = mergeHomeTiles({ recent_matches_has_more: false } as unknown as HomeTilesHead, [])
    expect(r.tiles).toEqual([])
  })
})

describe('homeMatchesUrl', () => {
  it('suite : curseur + taille de page', () => {
    expect(homeMatchesUrl('chief', { before: 'x_y' })).toBe(
      `/players/chief/pages/home/matches?before=x_y&limit=${HOME_TILES_PAGE_SIZE}`,
    )
  })
  it('curseur encodé', () => {
    expect(homeMatchesUrl('chief', { before: '2026-10-20T21:00:00Z_m 1' })).toContain(
      'before=2026-10-20T21%3A00%3A00Z_m+1',
    )
  })
  it('favoris : kind seul', () => {
    expect(homeMatchesUrl('chief', { favorites: true })).toBe('/players/chief/pages/home/matches?kind=favorites')
  })
})

describe('patchFavoriteFlag', () => {
  it('tête de l\'accueil : le drapeau de la tuile change, les autres sont intactes', () => {
    const data = { hero: 1, recent_matches: [tile('a'), tile('b')] }
    const out = patchFavoriteFlag(data, 'b', true)
    expect(out.recent_matches.map((m) => m.is_favorite)).toEqual([false, true])
    expect(out.recent_matches[0]).toBe(data.recent_matches[0])
    expect(out.hero).toBe(1)
  })
  it('page de tuiles et favoris (items)', () => {
    const out = patchFavoriteFlag({ items: [tile('a', true)], has_more: false }, 'a', false)
    expect(out.items[0].is_favorite).toBe(false)
  })
  it('rien à changer : MÊME référence (pas de rendu)', () => {
    const data = { items: [tile('a', true)] }
    expect(patchFavoriteFlag(data, 'a', true)).toBe(data)
    expect(patchFavoriteFlag(data, 'absent', true)).toBe(data)
  })
  it('formes inconnues ou absentes : inchangées', () => {
    expect(patchFavoriteFlag(undefined, 'a', true)).toBeUndefined()
    const other = { foo: 1 }
    expect(patchFavoriteFlag(other, 'a', true)).toBe(other)
  })
})

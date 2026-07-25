/**
 * Tests — invariance des query keys locale-dependantes (defis / season pass).
 *
 * Contexte (item backlog [i18n]) : le backend baque les libelles localises (titres
 * de defis, nom du pass, map/mode) dans le payload selon le header X-LevelUp-Locale
 * a l'instant du fetch. Si la locale n'entre PAS dans la query key, un switch de
 * langue laisse le cache (y compris le fetch background prefetch/poll) baken dans
 * l'ancienne langue. Ce test verrouille l'invariant : deux locales -> deux cles
 * distinctes (donc refetch naturel a la bascule), locale identique -> cle stable.
 */
import { describe, it, expect } from 'vitest'
import { queryKeys } from './keys'

describe('query keys locale-aware (bascule de langue -> refetch naturel)', () => {
  describe('home (defis de l accueil)', () => {
    it('FR vs EN : cles distinctes (le switch de langue change la cle)', () => {
      const fr = queryKeys.home('player-1', 'halo_infinite', 'fr')
      const en = queryKeys.home('player-1', 'halo_infinite', 'en')
      expect(fr).not.toEqual(en)
      expect(en).toContain('en')
      expect(fr).toContain('fr')
    })

    it('meme locale : cle stable (cache reutilise)', () => {
      expect(queryKeys.home('player-1', 'halo_infinite', 'fr')).toEqual(
        queryKeys.home('player-1', 'halo_infinite', 'fr'),
      )
    })

    it('la locale est le dernier segment de la cle', () => {
      expect(queryKeys.home('p', 'halo_5', 'en')).toEqual(['home', 'p', 'halo_5', 'en'])
    })
  })

  describe('seasonPass (defis / libelles du pass)', () => {
    it('FR vs EN : cles distinctes', () => {
      const fr = queryKeys.seasonPass('player-1', 'halo_infinite', 'fr')
      const en = queryKeys.seasonPass('player-1', 'halo_infinite', 'en')
      expect(fr).not.toEqual(en)
    })

    it('forme attendue avec titleSlug puis locale en dernier segment', () => {
      expect(queryKeys.seasonPass('p', 'halo_infinite', 'en')).toEqual([
        'palmares',
        'p',
        'halo_infinite',
        'season-pass',
        'en',
      ])
    })
  })
})

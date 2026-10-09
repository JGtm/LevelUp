/**
 * Tests unitaires — LeaderboardBlock.logic.
 *
 * La BORNE exacte du seuil (25 %) se teste ici plutôt qu'au rendu : une part pile à
 * 25 % demande un jeu de lignes précis, et le rendu ne dit pas de quel côté de la
 * borne on est tombé.
 */
import { describe, it, expect } from 'vitest'
import {
  ENRICHED_SORT_KEYS,
  isEnrichedSortKey,
  pickEffectiveOption,
  playlistsForSeason,
  resolveSort,
  showEnrichedColumns,
} from './LeaderboardBlock.logic'

/** Construit `total` lignes dont `enriched` portent des stats détaillées. */
function rows(enriched: number, total: number) {
  return Array.from({ length: total }, (_, i) => (i < enriched ? { match_count: 12 } : { match_count: null }))
}

describe('showEnrichedColumns', () => {
  it('table vide : aucune colonne', () => {
    expect(showEnrichedColumns([])).toBe(false)
  })

  it('sous le seuil (24 %) : colonnes masquées', () => {
    expect(showEnrichedColumns(rows(24, 100))).toBe(false)
  })

  it('pile au seuil (25 %) et au-delà : colonnes affichées', () => {
    expect(showEnrichedColumns(rows(1, 4))).toBe(true)
    expect(showEnrichedColumns(rows(79, 100))).toBe(true)
  })
})

describe('playlistsForSeason', () => {
  const options = [{ value: 'a' }, { value: 'b' }, { value: 'c' }]

  it('restreint aux playlists relevées pour la saison', () => {
    expect(playlistsForSeason(options, ['b', 'c'])).toEqual([{ value: 'b' }, { value: 'c' }])
  })

  it('champ absent (backend antérieur) : liste complète', () => {
    expect(playlistsForSeason(options, undefined)).toEqual(options)
    expect(playlistsForSeason(options, null)).toEqual(options)
    expect(playlistsForSeason(options, [])).toEqual(options)
  })

  it('aucune correspondance : liste complète plutôt qu’un sélecteur vide', () => {
    expect(playlistsForSeason(options, ['inconnue'])).toEqual(options)
  })
})

describe('isEnrichedSortKey', () => {
  it('reconnaît les clés portées par les colonnes enrichies', () => {
    for (const key of ENRICHED_SORT_KEYS) {
      expect(isEnrichedSortKey(key)).toBe(true)
    }
  })

  it('exclut les colonnes toujours affichées et celles des catégories de stats', () => {
    for (const key of ['rank', 'csr', 'matches', 'value', '']) {
      expect(isEnrichedSortKey(key)).toBe(false)
    }
  })
})

describe('resolveSort', () => {
  it('colonnes enrichies affichées : le tri demandé s’applique tel quel', () => {
    expect(resolveSort('kills', 'desc', true)).toEqual({ key: 'kills', dir: 'desc' })
  })

  it('colonnes enrichies masquées : un tri enrichi retombe sur le rang croissant', () => {
    // Sans ce repli, la table reste triée par une colonne invisible dont
    // `sortValue` rend 0 sur les lignes non enrichies : ordre incompréhensible,
    // et inannulable puisque l'en-tête qui portait le tri a disparu.
    expect(resolveSort('kills', 'desc', false)).toEqual({ key: 'rank', dir: 'asc' })
    expect(resolveSort('dmg_per_death', 'asc', false)).toEqual({ key: 'rank', dir: 'asc' })
  })

  it('colonnes enrichies masquées : un tri sur une colonne visible est préservé', () => {
    expect(resolveSort('csr', 'desc', false)).toEqual({ key: 'csr', dir: 'desc' })
    expect(resolveSort('rank', 'asc', false)).toEqual({ key: 'rank', dir: 'asc' })
    expect(resolveSort('value', 'desc', false)).toEqual({ key: 'value', dir: 'desc' })
  })

  it('ne modifie que le tri EFFECTIF : la valeur d’entrée est rendue intacte au retour des colonnes', () => {
    const chosen = { key: 'win_rate', dir: 'desc' } as const
    expect(resolveSort(chosen.key, chosen.dir, false)).toEqual({ key: 'rank', dir: 'asc' })
    expect(resolveSort(chosen.key, chosen.dir, true)).toEqual({ key: 'win_rate', dir: 'desc' })
  })
})

describe('pickEffectiveOption', () => {
  it('préserve le choix courant s’il figure dans les options', () => {
    expect(pickEffectiveOption([{ value: 'a' }, { value: 'b' }], 'b')).toBe('b')
  })

  it('retombe sur la première option quand le choix n’existe plus', () => {
    expect(pickEffectiveOption([{ value: 'a' }, { value: 'b' }], 'z')).toBe('a')
  })

  it('aucune option : conserve la valeur courante (pas de sélecteur vidé)', () => {
    expect(pickEffectiveOption([], 'z')).toBe('z')
  })
})

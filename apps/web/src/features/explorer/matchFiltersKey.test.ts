/**
 * matchFiltersKey.test.ts — la clé de cache Explorer doit BOUGER quand un filtre bouge.
 *
 * Ce test existe pour un mode de panne muet : un filtre ajouté au corps de la requête mais
 * pas à la clé ne casse rien à la compilation, ne lève aucune erreur à l'exécution, et rend
 * simplement le filtre inopérant à l'écran — TanStack Query resert l'entrée précédente. Le
 * cas constaté est `replay_scope` (lot A du backlog Notion), corrigé au lot F.
 */
import { describe, it, expect } from 'vitest'

import { matchFiltersKeyOf } from './queries'
import type { ExplorerMatchesQueryRequest } from '@/lib/api/types'

const base: ExplorerMatchesQueryRequest = {}

describe('matchFiltersKeyOf', () => {
  it('distingue deux requêtes qui ne diffèrent que par replay_scope', () => {
    const withReplay = matchFiltersKeyOf({ ...base, replay_scope: 'with' })
    const withoutReplay = matchFiltersKeyOf({ ...base, replay_scope: 'without' })
    const anyReplay = matchFiltersKeyOf(base)

    expect(withReplay).not.toBe(withoutReplay)
    expect(withReplay).not.toBe(anyReplay)
    expect(withoutReplay).not.toBe(anyReplay)
  })

  it('distingue deux requêtes qui ne diffèrent que par squad_scope', () => {
    expect(matchFiltersKeyOf({ ...base, squad_scope: 'solo' })).not.toBe(
      matchFiltersKeyOf({ ...base, squad_scope: 'squad' }),
    )
  })

  it('rend la MÊME clé quand seul l\'ordre de sélection change', () => {
    expect(matchFiltersKeyOf({ ...base, playlists: ['Ranked Arena', 'Big Team Battle'] })).toBe(
      matchFiltersKeyOf({ ...base, playlists: ['Big Team Battle', 'Ranked Arena'] }),
    )
  })

  it('un filtre vide et un filtre absent sont la même clé', () => {
    expect(matchFiltersKeyOf({ ...base, replay_scope: '' })).toBe(matchFiltersKeyOf(base))
  })
})

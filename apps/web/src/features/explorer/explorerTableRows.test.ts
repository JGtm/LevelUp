import { describe, it, expect } from 'vitest'
import { splitCommonMatchRows } from './explorerTableRows'
import type { ExplorerMatchesQueryResponse } from '@/lib/api/types'

type Ligne = NonNullable<ExplorerMatchesQueryResponse['table']['items']>[number]

function ligne(matchId: string): Ligne {
  return { match_id: matchId, map_ui: null, mode_ui: 'Slayer', playlist_label: null } as unknown as Ligne
}

describe('splitCommonMatchRows', () => {
  it("répartit les lignes par rôle en gardant l'ordre serveur", () => {
    const items = [ligne('m4'), ligne('m3'), ligne('m2'), ligne('m1')]
    const { ally, enemy } = splitCommonMatchRows(items, ['m1', 'm3'], ['m2', 'm4'])
    expect(ally.map((r) => r.match_id)).toEqual(['m3', 'm1'])
    expect(enemy.map((r) => r.match_id)).toEqual(['m4', 'm2'])
  })

  it('normalise les libellés nuls comme normalizeExplorerTableRows', () => {
    const { ally } = splitCommonMatchRows([ligne('m1')], ['m1'], [])
    expect(ally[0]).toMatchObject({ map_ui: '', mode_ui: 'Slayer', playlist_label: '' })
  })

  it("ignore une ligne d'un match qui n'est d'aucun rôle", () => {
    const { ally, enemy } = splitCommonMatchRows([ligne('mx')], ['m1'], ['m2'])
    expect(ally).toEqual([])
    expect(enemy).toEqual([])
  })
})

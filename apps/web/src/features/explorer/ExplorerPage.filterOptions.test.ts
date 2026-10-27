import { describe, expect, it } from 'vitest'
import type { ExplorerMatchesQueryResponse } from '@/lib/api/types'
import { buildExplorerFilterOptions } from './ExplorerPage.filterOptions'

const t = (key: string) => key

type Summary = NonNullable<ExplorerMatchesQueryResponse['summary']>

// Le serveur sert, pour chaque playlist, mode et carte, un libellé dans la langue de la requête et une
// valeur qui est le nom de la langue par défaut (clé de filtre `playlists`, `modes`, `maps`).
function resume(labels: { playlist: string; mode: string; map: string }): Summary {
  return {
    total_matches: 1,
    selected_match_id: null,
    available_playlists: [{ label: labels.playlist, value: 'Partie rapide', count: 0 }],
    available_modes: [{ label: labels.mode, value: 'Assassin', count: 0 }],
    available_maps: [{ label: labels.map, value: 'Ruelles', count: 0 }],
  } as Summary
}

describe('buildExplorerFilterOptions — libellé suit la langue, valeur ne bouge pas', () => {
  it('français : libellé = valeur', () => {
    const o = buildExplorerFilterOptions(resume({ playlist: 'Partie rapide', mode: 'Assassin', map: 'Ruelles' }), t)
    expect(o.playlistOptions).toEqual([{ value: 'Partie rapide', label: 'Partie rapide' }])
    expect(o.modeOptions).toEqual([{ value: 'Assassin', label: 'Assassin' }])
    expect(o.mapOptions).toEqual([{ value: 'Ruelles', label: 'Ruelles' }])
  })

  it('anglais : libellé anglais, valeur inchangée (le filtre `?maps=` garde la même clé)', () => {
    const o = buildExplorerFilterOptions(resume({ playlist: 'Quick Play', mode: 'Slayer', map: 'Streets' }), t)
    expect(o.playlistOptions).toEqual([{ value: 'Partie rapide', label: 'Quick Play' }])
    expect(o.modeOptions).toEqual([{ value: 'Assassin', label: 'Slayer' }])
    expect(o.mapOptions).toEqual([{ value: 'Ruelles', label: 'Streets' }])
  })

  it('sans résumé : listes vides', () => {
    const o = buildExplorerFilterOptions(undefined, t)
    expect(o.playlistOptions).toEqual([])
    expect(o.modeOptions).toEqual([])
    expect(o.mapOptions).toEqual([])
  })
})

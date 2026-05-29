import { describe, it, expect } from 'vitest'

import {
  DEFAULT_SORT_KEY,
  decodeExplorerScope,
  encodeExplorerScope,
  explorerSearchSchema,
  type ExplorerScope,
} from './explorerScope'

const fullScope: ExplorerScope = {
  startDate: '2026-04-01',
  endDate: '2026-05-01',
  squadScope: 'squad',
  matchIDSearch: 'abc-123',
  expTypes: new Set(['PVP classé']),
  playlists: new Set(['Ranked Arena', 'Big Team Battle']),
  mapNames: new Set(['Aquarius']),
  modeNames: new Set(['Slayer']),
  perfTiers: new Set(['4', '5']),
  skillTiers: new Set(['Onyx']),
  outcomeFilter: new Set(['2', '3']),
  sortKey: 'kda:desc',
}

describe('encode/decode round-trip', () => {
  it('un scope plein survit à encode → decode sans perte', () => {
    expect(decodeExplorerScope(encodeExplorerScope(fullScope))).toEqual(fullScope)
  })

  it('un objet vide décode vers les défauts', () => {
    const def = decodeExplorerScope({})
    expect(def).toEqual({
      startDate: '',
      endDate: '',
      squadScope: '',
      matchIDSearch: '',
      expTypes: new Set(),
      playlists: new Set(),
      mapNames: new Set(),
      modeNames: new Set(),
      perfTiers: new Set(),
      skillTiers: new Set(),
      outcomeFilter: new Set(),
      sortKey: DEFAULT_SORT_KEY,
    })
  })
})

describe('encodeExplorerScope', () => {
  it('omet les valeurs vides (URL propre)', () => {
    const encoded = encodeExplorerScope(decodeExplorerScope({}))
    // Toutes les clés doivent être undefined → JSON.stringify les retire.
    expect(JSON.stringify(encoded)).toBe('{}')
  })

  it('omet le tri par défaut mais conserve un tri custom', () => {
    expect(encodeExplorerScope({ ...fullScope, sortKey: DEFAULT_SORT_KEY }).sort).toBeUndefined()
    expect(encodeExplorerScope({ ...fullScope, sortKey: 'kills:desc' }).sort).toBe('kills:desc')
  })

  it('sérialise les Sets en csv', () => {
    expect(encodeExplorerScope(fullScope).pl).toBe('Ranked Arena,Big Team Battle')
    expect(encodeExplorerScope(fullScope).perf).toBe('4,5')
  })
})

describe('decodeExplorerScope', () => {
  it('ignore un squadScope invalide', () => {
    expect(decodeExplorerScope({ scope: 'bogus' as never }).squadScope).toBe('')
    expect(decodeExplorerScope({ scope: 'solo' }).squadScope).toBe('solo')
  })
})

describe('explorerSearchSchema (validateSearch)', () => {
  it('accepte un search complet', () => {
    const parsed = explorerSearchSchema.parse({
      mode: 'matches',
      pl: 'Ranked Arena',
      start: '2026-04-01',
      sort: 'kda:desc',
    })
    expect(parsed.pl).toBe('Ranked Arena')
    expect(parsed.mode).toBe('matches')
  })

  it('rejette un mode hors enum', () => {
    expect(() => explorerSearchSchema.parse({ mode: 'bogus' })).toThrow()
  })

  it('rejette un scope hors enum', () => {
    expect(() => explorerSearchSchema.parse({ scope: 'bogus' })).toThrow()
  })

  it('un search vide est valide (tous optionnels)', () => {
    expect(explorerSearchSchema.parse({})).toEqual({})
  })
})

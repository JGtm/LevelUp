import { describe, it, expect } from 'vitest'

import {
  decodeExplorerScope,
  encodeExplorerScope,
  explorerScopeToFilterSpec,
  explorerSearchSchema,
  type ExplorerScope,
} from './explorerScope'

const fullScope: ExplorerScope = {
  startDate: '2026-04-01',
  endDate: '2026-05-01',
  squadScope: 'squad',
  replayScope: 'with',
  matchIDSearch: 'abc-123',
  expTypes: new Set(['PVP classé']),
  playlists: new Set(['Ranked Arena', 'Big Team Battle']),
  mapNames: new Set(['Aquarius']),
  modeNames: new Set(['Slayer']),
  perfTiers: new Set(['4', '5']),
  skillTiers: new Set(['Onyx']),
  outcomeFilter: new Set(['2', '3']),
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
      replayScope: '',
      matchIDSearch: '',
      expTypes: new Set(),
      playlists: new Set(),
      mapNames: new Set(),
      modeNames: new Set(),
      perfTiers: new Set(),
      skillTiers: new Set(),
      outcomeFilter: new Set(),
    })
  })
})

describe('encodeExplorerScope', () => {
  it('omet les valeurs vides (URL propre)', () => {
    const encoded = encodeExplorerScope(decodeExplorerScope({}))
    // Toutes les clés doivent être undefined → JSON.stringify les retire.
    expect(JSON.stringify(encoded)).toBe('{}')
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

  it('ignore un replayScope invalide et accepte les 3 états', () => {
    expect(decodeExplorerScope({ replay: 'bogus' as never }).replayScope).toBe('')
    expect(decodeExplorerScope({}).replayScope).toBe('')
    expect(decodeExplorerScope({ replay: 'with' }).replayScope).toBe('with')
    expect(decodeExplorerScope({ replay: 'without' }).replayScope).toBe('without')
  })
})

describe('replayScope (filtre « Rejeu »)', () => {
  it('encode le scope rejeu dans le param `replay`, et l’omet quand il est vide', () => {
    const base = decodeExplorerScope({})
    expect(encodeExplorerScope({ ...base, replayScope: 'without' }).replay).toBe('without')
    expect(encodeExplorerScope(base).replay).toBeUndefined()
  })

  it('survit au round-trip encode → decode', () => {
    const s = { ...decodeExplorerScope({}), replayScope: 'with' as const }
    expect(decodeExplorerScope(encodeExplorerScope(s)).replayScope).toBe('with')
  })

  it('rejette un `replay` hors enum au niveau du schéma de route', () => {
    expect(() => explorerSearchSchema.parse({ replay: 'bogus' })).toThrow()
    expect(explorerSearchSchema.parse({ replay: 'with' }).replay).toBe('with')
  })
})

describe('explorerScopeToFilterSpec (Phase 4)', () => {
  const base = decodeExplorerScope({}) // tous filtres vides + défauts

  it('scope vide → undefined (pas de filterSpec, fallback Q25 global)', () => {
    expect(explorerScopeToFilterSpec(base)).toBeUndefined()
  })

  it('playlists multi → playlist_names', () => {
    expect(
      explorerScopeToFilterSpec({ ...base, playlists: new Set(['Ranked Arena', 'BTB']) }),
    ).toEqual({ playlist_names: ['Ranked Arena', 'BTB'] })
  })

  it('modeNames → mode_categories', () => {
    expect(explorerScopeToFilterSpec({ ...base, modeNames: new Set(['Fiesta']) })).toEqual({
      mode_categories: ['Fiesta'],
    })
  })

  it('dates → date_from/date_to (bornes inclusives)', () => {
    expect(
      explorerScopeToFilterSpec({ ...base, startDate: '2026-04-01', endDate: '2026-05-01' }),
    ).toEqual({
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-05-01T23:59:59Z',
    })
  })

  it('outcome unique (code → label) ; multi-outcome ignoré', () => {
    expect(explorerScopeToFilterSpec({ ...base, outcomeFilter: new Set(['2']) })).toEqual({
      outcome: 'win',
    })
    // 2 outcomes sélectionnés → pas de filtre outcome (mono-valeur côté spec)
    expect(explorerScopeToFilterSpec({ ...base, outcomeFilter: new Set(['2', '3']) })).toBeUndefined()
  })

  it('combinaison playlists + dates + outcome', () => {
    expect(
      explorerScopeToFilterSpec({
        ...base,
        playlists: new Set(['Ranked Arena']),
        startDate: '2026-04-01',
        outcomeFilter: new Set(['3']),
      }),
    ).toEqual({
      playlist_names: ['Ranked Arena'],
      date_from: '2026-04-01T00:00:00Z',
      outcome: 'loss',
    })
  })
})

describe('explorerSearchSchema (validateSearch)', () => {
  it('accepte un search complet', () => {
    const parsed = explorerSearchSchema.parse({
      mode: 'matches',
      pl: 'Ranked Arena',
      start: '2026-04-01',
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

/**
 * Tests tokenHealthDisplay — réduction des labels de source de credentials
 * (chip Source de la section Santé des tokens).
 */
import { describe, expect, it } from 'vitest'

import { credentialSourceParts, hasLegacyCredentialSource, TOKEN_ERROR_KEY } from './tokenHealthDisplay'

describe('credentialSourceParts', () => {
  it('réduit les labels watcher_* en "store" dédupliqué', () => {
    expect(credentialSourceParts('watcher_oauth')).toEqual(['store'])
    expect(credentialSourceParts('watcher_oauth+watcher_oauth')).toEqual(['store'])
  })

  it('laisse passer un label non-store tel quel (garde-rail visuel ADR 0023 Phase 5)', () => {
    // Ces labels ne sont plus produits par le back : s'ils réapparaissent, ils
    // doivent rester VISIBLES (et flaggés legacy), pas être maquillés.
    expect(credentialSourceParts('duckdb_oauth')).toEqual(['duckdb_oauth'])
    expect(credentialSourceParts('env_oauth')).toEqual(['env_oauth'])
    expect(credentialSourceParts('future_source')).toEqual(['future_source'])
  })

  it('combine store + résidu inattendu sans dédupliquer à tort', () => {
    expect(credentialSourceParts('watcher_oauth+duckdb_oauth')).toEqual(['store', 'duckdb_oauth'])
  })
})

describe('hasLegacyCredentialSource', () => {
  it('false pour un store pur, true dès qu\'une autre source apparaît', () => {
    expect(hasLegacyCredentialSource(['store'])).toBe(false)
    expect(hasLegacyCredentialSource(['store', 'duckdb_oauth'])).toBe(true)
    expect(hasLegacyCredentialSource(['env_oauth'])).toBe(true)
  })
})

describe('TOKEN_ERROR_KEY', () => {
  it('couvre les 3 classes du backend (config/revoked/transient)', () => {
    expect(Object.keys(TOKEN_ERROR_KEY).sort()).toEqual(['config', 'revoked', 'transient'])
  })
})

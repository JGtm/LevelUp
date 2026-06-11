/**
 * Tests tokenHealthDisplay — réduction des labels de source de credentials
 * (chip Source de la section Santé des tokens).
 */
import { describe, expect, it } from 'vitest'

import { credentialSourceParts, hasLegacyCredentialSource, TOKEN_ERROR_KEY } from './tokenHealthDisplay'

describe('credentialSourceParts', () => {
  it('réduit les labels watcher_* en "store" dédupliqué', () => {
    expect(credentialSourceParts('watcher_msal+watcher_oauth')).toEqual(['store'])
    expect(credentialSourceParts('watcher_oauth')).toEqual(['store'])
  })

  it('mappe les fallbacks legacy en familles courtes', () => {
    expect(credentialSourceParts('duckdb_msal+env_oauth')).toEqual(['sync_meta', 'env'])
    expect(credentialSourceParts('watcher_legacy')).toEqual(['legacy'])
  })

  it('combine store + résidus legacy sans dédupliquer à tort', () => {
    expect(credentialSourceParts('watcher_oauth+duckdb_msal')).toEqual(['store', 'sync_meta'])
  })

  it('laisse passer un label inconnu tel quel (forward-compat)', () => {
    expect(credentialSourceParts('future_source')).toEqual(['future_source'])
  })
})

describe('hasLegacyCredentialSource', () => {
  it('false pour un store pur, true dès qu\'un fallback est présent', () => {
    expect(hasLegacyCredentialSource(['store'])).toBe(false)
    expect(hasLegacyCredentialSource(['store', 'sync_meta'])).toBe(true)
    expect(hasLegacyCredentialSource(['env'])).toBe(true)
  })
})

describe('TOKEN_ERROR_KEY', () => {
  it('couvre les 3 classes du backend (config/revoked/transient)', () => {
    expect(Object.keys(TOKEN_ERROR_KEY).sort()).toEqual(['config', 'revoked', 'transient'])
  })
})

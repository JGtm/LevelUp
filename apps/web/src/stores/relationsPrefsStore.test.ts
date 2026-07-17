import { describe, expect, it } from 'vitest'

import { migrateRelationsPrefs } from './relationsPrefsStore'

describe('migrateRelationsPrefs', () => {
  it('v1 → v2 : réinitialise includeNeverFaced à false, préserve filter/heatmapMode', () => {
    const v1 = { filter: 'rivals', includeFriends: true, heatmapMode: 'day' }
    const out = migrateRelationsPrefs(v1, 1)
    expect(out.includeNeverFaced).toBe(false)
    expect(out.filter).toBe('rivals')
    expect(out.heatmapMode).toBe('day')
    // L'ancienne clé de sémantique trompeuse est supprimée.
    expect((out as unknown as Record<string, unknown>).includeFriends).toBeUndefined()
  })

  it('v0 → v2 : daypart → hour ET includeNeverFaced défaut false', () => {
    const v0 = { filter: 'all', heatmapMode: 'daypart', includeFriends: false }
    const out = migrateRelationsPrefs(v0, 0)
    expect(out.heatmapMode).toBe('hour')
    expect(out.includeNeverFaced).toBe(false)
    expect((out as unknown as Record<string, unknown>).includeFriends).toBeUndefined()
  })

  it('state persisté vide / indéfini → défauts sûrs', () => {
    const out = migrateRelationsPrefs(undefined, 1)
    expect(out.includeNeverFaced).toBe(false)
  })
})

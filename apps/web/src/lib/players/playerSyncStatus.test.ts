/**
 * Tests — délai d'interrogation de l'état de synchronisation d'un joueur.
 */
import { describe, expect, it } from 'vitest'
import type { PlayerSyncStatus } from '@/lib/api/types'
import { syncStatusPollInterval } from './playerSyncStatus'

function status(over: Partial<PlayerSyncStatus>): PlayerSyncStatus {
  return { player_slug: 'p1', syncing: false, cooldown_retry_after_s: 0, history_complete: false, ...over }
}

describe('syncStatusPollInterval', () => {
  it('ne sonde pas tant que rien n’est arrivé', () => {
    expect(syncStatusPollInterval(undefined, false)).toBe(false)
    expect(syncStatusPollInterval(undefined, true)).toBe(false)
  })

  it('sonde toutes les 3 s tant qu’une synchronisation tourne, avec ou sans sondage au repos', () => {
    expect(syncStatusPollInterval(status({ syncing: true }), false)).toBe(3_000)
    expect(syncStatusPollInterval(status({ syncing: true }), true)).toBe(3_000)
  })

  it('sonde toutes les 15 s pendant un délai', () => {
    expect(syncStatusPollInterval(status({ cooldown_retry_after_s: 120 }), false)).toBe(15_000)
  })

  it('au repos : aucun sondage par défaut (accueil), 15 s à la demande (carte des réglages)', () => {
    expect(syncStatusPollInterval(status({}), false)).toBe(false)
    expect(syncStatusPollInterval(status({}), true)).toBe(15_000)
  })
})

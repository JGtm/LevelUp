/**
 * waypointUrl.test — contrat du helper canonique d'URL Halo Waypoint (I19).
 * Formats vérifiés par l'utilisateur (2026-07-24) : segment par titre + bucket
 * « arena » pour Halo 5.
 */
import { describe, it, expect } from 'vitest'
import { buildWaypointMatchUrl } from './waypointUrl'

describe('buildWaypointMatchUrl', () => {
  it('Halo Infinite (défaut) : segment halo-infinite sans bucket', () => {
    expect(buildWaypointMatchUrl('JGtm', 'm-1')).toBe(
      'https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/m-1',
    )
    expect(buildWaypointMatchUrl('JGtm', 'm-1', 'halo_infinite')).toBe(
      'https://www.halowaypoint.com/halo-infinite/players/JGtm/matches/m-1',
    )
  })

  it('Halo 5 : segment halo-5-guardians + bucket arena', () => {
    expect(buildWaypointMatchUrl('JGtm', '5d16ff8d-43df-4300-8c87-ed83b03674d2', 'halo_5')).toBe(
      'https://www.halowaypoint.com/halo-5-guardians/players/JGtm/matches/arena/5d16ff8d-43df-4300-8c87-ed83b03674d2',
    )
  })

  it('gamertag avec espace : encodé', () => {
    expect(buildWaypointMatchUrl('Gamer Tag', 'm-2')).toContain('/players/Gamer%20Tag/')
  })
})

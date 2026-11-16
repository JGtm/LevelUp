/**
 * filmLayerAvailability.test.ts — la table de ce que le film porte, rangée sous les clés du panneau :
 * un compte nul n'ouvre pas son calque, un calque indisponible reste indisponible, la probabilité de
 * réapparition est rangée à sa clé.
 */
import { describe, expect, it } from 'vitest'

import { filmLayerAvailability, type FilmLayerSources } from './filmLayerAvailability'

const ON = { available: true }
const OFF = { available: false }

const FULL: FilmLayerSources = {
  zoneCount: 3,
  placements: { counts: { drawable: 4, unnamed: 1, dropped: 2 } },
  weaponPads: ON, groundWeapons: ON, objectiveCount: 2, flags: ON, vipCrown: ON, skullCarrier: ON, bombCarrier: ON, vehicles: ON,
  hasWallPlan: true,
  lives: { spawns: true, deaths: true, killers: false, followedKnown: true },
  respawnForecast: ON,
}

describe('filmLayerAvailability', () => {
  it('un film complet ouvre tous les calques', () => {
    expect(filmLayerAvailability(FULL)).toEqual({
      zones: true,
      placements: { drawable: true, unnamed: true, dropped: true },
      weaponPads: true, groundWeapons: true, modeObjectives: true, flagCarries: true, vipCrown: true, skullCarrier: true, bombCarrier: true,
      vehicles: true, wallPlan: true,
      lives: { spawns: true, deaths: true, killers: false, followedKnown: true },
      respawnForecast: true,
    })
  })

  it('un compte nul n’ouvre pas son calque', () => {
    const a = filmLayerAvailability({ ...FULL, zoneCount: 0, objectiveCount: 0, placements: { counts: { drawable: 0, unnamed: 0, dropped: 0 } }, hasWallPlan: false })
    expect([a.zones, a.modeObjectives, a.wallPlan, a.placements]).toEqual([false, false, false, { drawable: false, unnamed: false, dropped: false }])
  })

  it('chaque hook garde sa propre clé : couper l’un ne touche pas les autres', () => {
    const a = filmLayerAvailability({ ...FULL, weaponPads: OFF, respawnForecast: OFF })
    expect([a.weaponPads, a.respawnForecast, a.groundWeapons, a.vehicles]).toEqual([false, false, true, true])
  })
})

import { describe, expect, it } from 'vitest'

import { startingWeaponKeys, visibleGroundWeapons } from './startingWeaponsOnGround'

const LOADOUTS = [
  { src: 'birth', w: ['0x48C19D2D', '0xF408190F'] },
  { src: 'kf', w: ['0xAAAAAAAA'] },
]
const spawnedDepart = { w: '48c19d2d', origin: 'spawned' as const }
const droppedDepart = { w: 'f408190f', origin: 'dropped' as const }
const droppedPuissance = { w: '2b1824d5', origin: 'dropped' as const }
const droppedNonNaissance = { w: 'aaaaaaaa', origin: 'dropped' as const }
const tout = [spawnedDepart, droppedDepart, droppedPuissance, droppedNonNaissance]

describe('visibleGroundWeapons', () => {
  it('garde toujours une arme posee par la carte, meme modele que le depart', () => {
    expect(visibleGroundWeapons([spawnedDepart], LOADOUTS, false)).toEqual([spawnedDepart])
  })
  it('masque une arme lachee qui est une arme de depart', () => {
    expect(visibleGroundWeapons([droppedDepart], LOADOUTS, false)).toEqual([])
  })
  it('garde une arme lachee de puissance', () => {
    expect(visibleGroundWeapons([droppedPuissance], LOADOUTS, false)).toEqual([droppedPuissance])
  })
  it('seules les dotations de naissance comptent', () => {
    expect(visibleGroundWeapons([droppedNonNaissance], LOADOUTS, false)).toEqual([droppedNonNaissance])
  })
  it('option activee : tout est garde', () => {
    expect(visibleGroundWeapons(tout, LOADOUTS, true)).toEqual(tout)
  })
  it('document sans dotation de naissance : tout est garde', () => {
    expect(visibleGroundWeapons(tout, [], false)).toEqual(tout)
    expect(visibleGroundWeapons(tout, undefined, false)).toEqual(tout)
    expect(visibleGroundWeapons(tout, [{ src: 'kf', w: ['0x48C19D2D'] }], false)).toEqual(tout)
  })
  it('normalise les identifiants des deux ecritures', () => {
    expect([...startingWeaponKeys(LOADOUTS)].sort()).toEqual(['0x48C19D2D', '0xF408190F'])
    expect(visibleGroundWeapons(tout, LOADOUTS, false)).toEqual([spawnedDepart, droppedPuissance, droppedNonNaissance])
  })
})

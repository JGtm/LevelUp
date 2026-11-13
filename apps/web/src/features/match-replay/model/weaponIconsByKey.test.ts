/**
 * Tests — weaponIconsByKey : le pictogramme d'une arme retrouvé par son weapon_key dans le fil.
 *
 * Ce qu'ils protègent : seule une ligne qui nomme une arme (clé) ET en sert un dessin entre dans la
 * table ; le premier dessin d'une clé est conservé ; une ligne sans clé ou sans image n'invente rien.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayFeedEntry } from './killFeedLogic'
import { weaponIconsByKey } from './weaponIconsByKey'

function killEntry(over: Record<string, unknown>): ReplayFeedEntry {
  return {
    key: 'k',
    replayMs: 0,
    medal: null,
    death: null,
    kill: { weaponKey: '', weaponLabel: '', weaponImageUrl: '', weaponTinted: false, ...over } as never,
  }
}

describe('weaponIconsByKey', () => {
  it('relève le dessin, la teinte et le nom de chaque arme du fil', () => {
    const icons = weaponIconsByKey([
      killEntry({ weaponKey: 'hinf_br75', weaponLabel: 'BR75', weaponImageUrl: '/br75.png', weaponTinted: true }),
    ])
    expect(icons.get('hinf_br75')).toEqual({ imageUrl: '/br75.png', tinted: true, label: 'BR75' })
  })

  it('écarte une ligne sans clé, sans image, ou qui n’est pas une élimination', () => {
    const icons = weaponIconsByKey([
      killEntry({ weaponKey: '', weaponImageUrl: '/melee.png' }),
      killEntry({ weaponKey: 'hinf_sans_image', weaponImageUrl: '' }),
      { key: 'm', replayMs: 0, kill: null, medal: null, death: null },
    ])
    expect(icons.size).toBe(0)
  })

  it('garde le premier dessin d’une clé', () => {
    const icons = weaponIconsByKey([
      killEntry({ weaponKey: 'hinf_x', weaponImageUrl: '/premier.png' }),
      killEntry({ weaponKey: 'hinf_x', weaponImageUrl: '/second.png' }),
    ])
    expect(icons.get('hinf_x')?.imageUrl).toBe('/premier.png')
  })

  it('un fil vide ne donne aucun dessin', () => {
    expect(weaponIconsByKey([]).size).toBe(0)
  })
})

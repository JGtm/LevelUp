import { describe, expect, it } from 'vitest'

import { VIES_PAR_DEFAUT, VIES_VALEURS, viesEffectives, viesPostees } from './vies.logic'

describe('vies.logic — les apparitions qui forment les grappes', () => {
  it('le défaut est « Départ » et ne se poste pas : le corps d’avant la bascule est conservé', () => {
    expect(VIES_PAR_DEFAUT).toBe('depart')
    expect(VIES_VALEURS).toEqual(['depart', 'toutes'])
    expect(viesPostees('depart')).toBeUndefined()
    expect(viesPostees('toutes')).toBe('toutes')
  })

  it('un titre sans `spawn_forecast` ne demande jamais « Toutes les vies »', () => {
    expect(viesEffectives('toutes', true)).toBe('toutes')
    expect(viesEffectives('toutes', false)).toBe('depart')
    expect(viesEffectives('depart', false)).toBe('depart')
  })
})

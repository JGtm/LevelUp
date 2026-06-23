import { describe, it, expect } from 'vitest'

import { substituteHpToken } from './effectiveHp'

describe('substituteHpToken', () => {
  it('remplace toutes les occurrences du jeton {{HP}} par le barème', () => {
    expect(substituteHpToken('1 vie ({{HP}})', 115)).toBe('1 vie (115)')
    expect(substituteHpToken('≈ {{HP}} points · {{HP}} PV', 225)).toBe('≈ 225 points · 225 PV')
  })
  it('laisse le texte intact sans jeton', () => {
    expect(substituteHpToken('aucun jeton', 115)).toBe('aucun jeton')
  })
})

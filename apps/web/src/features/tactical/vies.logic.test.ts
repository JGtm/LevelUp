import { describe, expect, it } from 'vitest'

import { VIES_PAR_DEFAUT, VIES_VALEURS, viesPostees, viesProposees } from './vies.logic'

describe('vies.logic — les apparitions qui forment les grappes', () => {
  it('le défaut est « Départ » et ne se poste pas : le corps d’avant la bascule est conservé', () => {
    expect(VIES_PAR_DEFAUT).toBe('depart')
    expect(VIES_VALEURS).toEqual(['depart', 'toutes'])
    expect(viesPostees('depart')).toBeUndefined()
    expect(viesPostees('toutes')).toBe('toutes')
  })

  it('la bascule est proposée avec des grappes, un filtre posé, un réglage à ôter ou une lecture d’artefact', () => {
    const base = { question: 'morts', nbGrappes: 0, spawn: '', vies: 'depart' } as const
    expect(viesProposees(base)).toBe(false)
    expect(viesProposees({ ...base, nbGrappes: 2 })).toBe(true)
    expect(viesProposees({ ...base, spawn: 'g1' })).toBe(true)
    expect(viesProposees({ ...base, vies: 'toutes' })).toBe(true)
    // Sous « Départ », une carte peut n'avoir aucune grappe alors qu'« Toutes les vies » en aurait.
    expect(viesProposees({ ...base, question: 'temps' })).toBe(true)
    expect(viesProposees({ ...base, question: 'routes' })).toBe(true)
  })
})

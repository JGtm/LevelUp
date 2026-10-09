import { describe, expect, it } from 'vitest'

import { piedDePageAffiche } from './piedDePage'

describe('piedDePageAffiche', () => {
  it('par défaut, la coquille pose le pied de page', () => {
    expect(piedDePageAffiche([{}, { staticData: {} }])).toBe(true)
  })

  it('une route de la chaîne qui le refuse suffit à le retirer (vue cockpit)', () => {
    expect(piedDePageAffiche([{ staticData: {} }, { staticData: { piedDePage: false } }])).toBe(false)
  })

  it('piedDePage: true ne force rien de plus que le défaut', () => {
    expect(piedDePageAffiche([{ staticData: { piedDePage: true } }])).toBe(true)
  })
})

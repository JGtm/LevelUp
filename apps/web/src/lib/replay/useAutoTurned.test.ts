/**
 * useAutoTurned.test.ts — L'ÉTAT DE LA RÈGLE AUTO : hystérésis, portée, gel, choix retenu.
 *
 * Les cotes sont celles d'une carte de 100 x 120 dans une boîte de hauteur 100 : le gain du plan tourné y vaut
 * ~30 % à 114 de large (au-dessus du seuil pour tourner), ~23 % à 111 (entre les deux seuils) et ~0 % à 100.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { QuarterTurns } from './orientation'
import { useAutoTurned, type AutoTurnedInput } from './useAutoTurned'

const PORTRAIT = { largeur: 100, hauteur: 120 }

function entree(largeur: number, reste: Partial<AutoTurnedInput> = {}): AutoTurnedInput {
  return { scopeKey: 'm1', choice: null, carte: PORTRAIT, boite: { largeur, hauteur: 100 }, ...reste }
}

function monter(initial: AutoTurnedInput) {
  return renderHook((p: AutoTurnedInput) => useAutoTurned(p), { initialProps: initial })
}

describe('useAutoTurned', () => {
  it('tourne au-dessus du seuil, garde le plan tourné entre les deux seuils, le redresse sous le seuil de retour', () => {
    const { result, rerender } = monter(entree(111))
    expect(result.current).toBe(false)
    rerender(entree(114))
    expect(result.current).toBe(true)
    rerender(entree(111))
    expect(result.current).toBe(true)
    rerender(entree(100))
    expect(result.current).toBe(false)
    rerender(entree(111))
    expect(result.current).toBe(false)
  })

  it('une autre portée repart du plan droit', () => {
    const { result, rerender } = monter(entree(114))
    expect(result.current).toBe(true)
    rerender(entree(111, { scopeKey: 'm2' }))
    expect(result.current).toBe(false)
  })

  it('un choix retenu l’emporte et efface l’état : le retour en Auto repart du plan droit', () => {
    const { result, rerender } = monter(entree(114))
    expect(result.current).toBe(true)
    for (const choice of [0, 1, 2, 3] as QuarterTurns[]) {
      rerender(entree(114, { choice }))
      expect(result.current, String(choice)).toBe(false)
    }
    rerender(entree(111))
    expect(result.current).toBe(false)
  })

  it('gelé : la sortie reste celle d’avant et rien n’est rangé ; dégelé, la règle reprend avec l’état d’avant', () => {
    const { result, rerender } = monter(entree(114))
    expect(result.current).toBe(true)
    rerender(entree(100, { gele: true }))
    expect(result.current).toBe(true)
    rerender(entree(111))
    expect(result.current).toBe(true)
  })

  it('une carte inconnue, ou une boîte pas mesurée : le plan droit', () => {
    expect(monter(entree(114, { carte: null })).result.current).toBe(false)
    expect(monter(entree(0)).result.current).toBe(false)
  })
})

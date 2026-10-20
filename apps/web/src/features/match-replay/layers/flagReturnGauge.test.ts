import { describe, expect, it } from 'vitest'

import { flagReturnAt, flagSpanAt } from './flagCarriesLayer'
import type { ReplayFlagCarryReady } from '@/lib/replay/replayNormalize'

/**
 * LA JAUGE DE RETOUR (schéma 63) — un ESCALIER, jamais une interpolation.
 *
 * Le film émet par paliers et la valeur TIENT jusqu'au point suivant : interpoler inventerait un
 * remplissage régulier que le jeu n'a pas (son taux suit une série harmonique) et masquerait les
 * REDESCENTES, qui sont le vidage quand plus personne n'est dans la zone.
 */
describe('flagReturnAt — la jauge de retour tenue en escalier', () => {
  const POINTS = [
    { t: 20, v: 0.02 },
    { t: 23, v: 0.5 },
    { t: 26, v: 0.2 },
    { t: 29, v: 1 },
  ]

  it('tient la dernière valeur jusqu’au point suivant, sans jamais interpoler', () => {
    expect(flagReturnAt(POINTS, 20), 'sur le premier point').toBe(0.02)
    expect(flagReturnAt(POINTS, 22), 'entre deux points : la valeur TENUE').toBe(0.02)
    expect(flagReturnAt(POINTS, 23)).toBe(0.5)
    expect(flagReturnAt(POINTS, 25), 'toujours tenue, pas 0,35').toBe(0.5)
  })

  it('suit la jauge qui SE VIDE — une redescente est une lecture, pas une anomalie', () => {
    expect(flagReturnAt(POINTS, 26)).toBe(0.2)
    expect(flagReturnAt(POINTS, 28)).toBe(0.2)
    expect(flagReturnAt(POINTS, 29)).toBe(1)
  })

  it('tient la dernière valeur APRÈS le dernier point : l’intervalle la porte jusqu’au bout', () => {
    expect(flagReturnAt(POINTS, 999)).toBe(1)
  })

  it('rend `null` AVANT le premier point, et sur l’absence — jamais zéro', () => {
    expect(flagReturnAt(POINTS, 19), 'avant le premier point : rien à dire').toBeNull()
    expect(flagReturnAt([], 25), 'tableau vide').toBeNull()
    expect(flagReturnAt(undefined, 25), 'artefact antérieur au schéma 63').toBeNull()
  })

  it('`flagSpanAt` la pose sur l’intervalle courant', () => {
    const carry: ReplayFlagCarryReady = {
      team: 0,
      spans: [
        { state: 'dropped', t0: 20, t1: 29, xuid: null, x: 5, y: 5, returnProgress: POINTS },
        { state: 'home', t0: 30, t1: 39, xuid: null, x: 1, y: 9, returnProgress: [] },
      ],
    }
    expect(flagSpanAt(carry, 25)?.returnProgress).toBe(0.5)
    expect(flagSpanAt(carry, 35)?.returnProgress, 'un drapeau rentré n’a pas de jauge').toBeNull()
  })
})

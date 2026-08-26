/**
 * replaySoundVariants.test.ts — le TIRAGE d'une variante.
 *
 * Ce que ces tests protègent : que le tirage soit UNIFORME et BORNÉ (un `Math.random()` qui
 * rendrait exactement 1 ne doit pas sortir de la liste), et qu'un geste à variantes soit
 * poussé AVEC ses variantes par le constructeur — sans quoi le préchargement ne les
 * connaîtrait pas et le tirage jouerait un fichier jamais décodé, c'est-à-dire un silence.
 */
import { describe, expect, it } from 'vitest'

import { pickVariantStem, soundEvent, SOUND_VARIANTS, stemsOf } from './replaySoundVariants'

describe('pickVariantStem — uniforme et borné', () => {
  const ev = { stem: 'grapple_fire', variants: ['a', 'b', 'c'] as const }

  it('tire la variante que le hasard désigne', () => {
    expect(pickVariantStem(ev, () => 0)).toBe('a')
    expect(pickVariantStem(ev, () => 0.5)).toBe('b')
    expect(pickVariantStem(ev, () => 0.99)).toBe('c')
  })

  it('un hasard à 1 (borne exclue en théorie) reste DANS la liste', () => {
    expect(pickVariantStem(ev, () => 1)).toBe('c')
  })

  it('sans variantes, le stem se joue tel quel', () => {
    expect(pickVariantStem({ stem: 'melee_kill' }, () => 0.9)).toBe('melee_kill')
    expect(pickVariantStem({ stem: 'melee_kill', variants: [] }, () => 0.9)).toBe('melee_kill')
  })

  it('les trois variantes sortent toutes sur un balayage du hasard', () => {
    const vus = new Set(
      Array.from({ length: 30 }, (_, i) => pickVariantStem(ev, () => i / 30)),
    )
    expect([...vus].sort()).toEqual(['a', 'b', 'c'])
  })
})

describe('soundEvent — un geste à variantes ne peut pas être poussé nu', () => {
  it('attache les variantes du manifeste', () => {
    expect(soundEvent(120, 'grapple_fire')).toEqual({
      ms: 120,
      stem: 'grapple_fire',
      variants: SOUND_VARIANTS.grapple_fire,
    })
  })

  it('laisse intact un stem sans variante', () => {
    expect(soundEvent(120, 'melee_kill')).toEqual({ ms: 120, stem: 'melee_kill' })
  })
})

describe('stemsOf — ce que le préchargement doit couvrir', () => {
  it('rend TOUTES les variantes, pas seulement la première', () => {
    expect(stemsOf(soundEvent(0, 'repulsor_kill'))).toEqual(SOUND_VARIANTS.repulsor_kill)
  })

  it('rend le seul stem quand il n y a pas de variante', () => {
    expect(stemsOf({ stem: 'camo_activate' })).toEqual(['camo_activate'])
  })

  it('la première variante EST le stem nu — les tables existantes le nomment déjà ainsi', () => {
    for (const [stem, variantes] of Object.entries(SOUND_VARIANTS)) {
      expect(variantes[0]).toBe(stem)
    }
  })
})

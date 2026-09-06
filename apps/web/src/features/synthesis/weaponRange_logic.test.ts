/**
 * weaponRange_logic.test — les décisions de lecture de la section « Portée par arme ».
 *
 * Ce que ces tests verrouillent : le repli d'un libellé non résolu sur la clé d'arme (un
 * trou de registre doit SE VOIR), l'ordre backend des armes sous le seuil (jamais rejoué
 * côté front), le `null` qui permet d'OMETTRE un demi-énoncé plutôt que d'écrire « frags : »
 * tout seul, et la garde qui empêche une carte à zéro ligne.
 */
import { describe, expect, it } from 'vitest'

import type { SynthesisWeaponRange, WeaponBelowThreshold } from '@/lib/api/types'

import { belowThresholdNames, hasWeaponRangeRows, resolveWeaponLabel } from './weaponRange_logic'

const BELOW: WeaponBelowThreshold[] = [
  { weapon_key: 'hinf_hydra', label: 'Hydra', label_en: 'Hydra', measured: 6 },
  { weapon_key: 'hinf_disruptor', label: 'Disrupteur', label_en: 'Disruptor', measured: 4 },
]

const count = (n: number) => String(n)

describe('resolveWeaponLabel', () => {
  it('rend le libellé de la locale demandée', () => {
    const w = { weapon_key: 'hinf_br75', label: 'Fusil de combat BR75', label_en: 'BR75 Battle Rifle' }
    expect(resolveWeaponLabel(w, 'fr')).toBe('Fusil de combat BR75')
    expect(resolveWeaponLabel(w, 'en')).toBe('BR75 Battle Rifle')
  })

  it('replie sur la clé d’arme quand le registre n’a rien résolu (libellé absent ou vide)', () => {
    expect(resolveWeaponLabel({ weapon_key: 'hinf_inconnue' }, 'fr')).toBe('hinf_inconnue')
    expect(resolveWeaponLabel({ weapon_key: 'hinf_inconnue', label: '  ' }, 'fr')).toBe('hinf_inconnue')
    // Locale EN sans `label_en` : le repli est la clé, JAMAIS le libellé français — servir
    // l'autre langue serait un mélange silencieux.
    expect(resolveWeaponLabel({ weapon_key: 'hinf_x', label: 'Hydra' }, 'en')).toBe('hinf_x')
  })
})

describe('belowThresholdNames', () => {
  it('nomme chaque arme avec son effectif, dans l’ordre du backend', () => {
    expect(belowThresholdNames(BELOW, 'fr', count)).toBe('Hydra (6), Disrupteur (4)')
  })

  it('rend null sur une liste vide ou absente — l’appelant OMET le demi-énoncé', () => {
    expect(belowThresholdNames([], 'fr', count)).toBeNull()
    expect(belowThresholdNames(undefined, 'fr', count)).toBeNull()
    expect(belowThresholdNames(null, 'fr', count)).toBeNull()
  })

  it('replie sur la clé d’arme comme le reste de la section', () => {
    expect(belowThresholdNames([{ weapon_key: 'hinf_x', measured: 2 }], 'fr', count)).toBe('hinf_x (2)')
  })
})

describe('hasWeaponRangeRows', () => {
  const bloc = (weapons: SynthesisWeaponRange['weapons']): SynthesisWeaponRange => ({
    weapons,
    median_kills_m: 7.4,
    median_deaths_m: 11.8,
    measured_kills: 12,
    total_kills: 20,
    measured_deaths: 10,
    total_deaths: 18,
  })

  it('vrai dès qu’une arme est publiable', () => {
    expect(hasWeaponRangeRows(bloc([{ weapon_key: 'hinf_br75' }]))).toBe(true)
  })

  it('faux sur un bloc sans arme, une liste nulle, ou pas de bloc du tout', () => {
    expect(hasWeaponRangeRows(bloc([]))).toBe(false)
    expect(hasWeaponRangeRows(bloc(null))).toBe(false)
    expect(hasWeaponRangeRows(null)).toBe(false)
    expect(hasWeaponRangeRows(undefined)).toBe(false)
  })
})

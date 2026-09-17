/**
 * weaponRange_logic.test — les décisions de lecture de la section « Portée par arme ».
 *
 * Ce que ces tests verrouillent : la garde qui empêche une carte à zéro ligne, et le miroir
 * du seuil Go. Le repli de libellé a suivi `resolveWeaponLabel` dans
 * `components/charts/weaponRangeChart.test.ts` le 2026-09-17.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import type { SynthesisWeaponRange } from '@/lib/api/types'

import { WEAPON_RANGE_MIN_MEASURED, hasWeaponRangeRows } from './weaponRange_logic'

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

/**
 * LE GARDE-RAIL INTER-LANGAGES DU SEUIL DE PUBLICATION.
 *
 * `WEAPON_RANGE_MIN_MEASURED` est un MIROIR de `analysis.WeaponRangeMinMeasured` (Go) : le
 * contrat ne sert pas le seuil, le front n'en a besoin que pour ÉCRIRE « sous le seuil de N
 * mesures ». Un miroir sans garde-rail dérive en silence — la phrase devient fausse sans
 * qu'aucun test ne bouge. Ce test LIT la source Go et compare.
 *
 * Il ÉCHOUE si le fichier Go est introuvable ou si la constante n'y est plus : un skip
 * silencieux transformerait ce garde-rail en décoration au premier renommage.
 */
describe('WEAPON_RANGE_MIN_MEASURED — miroir du seuil Go', () => {
  it('vaut exactement analysis.WeaponRangeMinMeasured', () => {
    // Deux racines possibles selon d'où vitest est lancé (`apps/web`, ou le dépôt) : on
    // essaie les deux et on ÉCHOUE si aucune ne répond — jamais de skip silencieux.
    const candidates = [
      resolve(process.cwd(), '../go-api/internal/analysis/weapon_range.go'),
      resolve(process.cwd(), 'apps/go-api/internal/analysis/weapon_range.go'),
    ]
    let go: string | null = null
    for (const source of candidates) {
      try {
        go = readFileSync(source, 'utf8')
        break
      } catch {
        // Chemin suivant : c'est le `throw` ci-dessous qui tranche si aucun ne répond.
      }
    }
    if (go === null) {
      throw new Error(
        `Source Go du seuil illisible (essayé : ${candidates.join(', ')}) — le garde-rail du ` +
          'miroir ne peut pas se prononcer. Corriger le chemin plutôt que supprimer le test.',
      )
    }
    const match = /WeaponRangeMinMeasured\s*=\s*(\d+)/.exec(go)
    expect(
      match,
      'Constante `WeaponRangeMinMeasured` introuvable dans weapon_range.go — renommée ou ' +
        'déplacée : mettre à jour ce garde-rail ET weaponRange_logic.ts.',
    ).not.toBeNull()
    const goValue = Number(match![1])
    expect(
      WEAPON_RANGE_MIN_MEASURED,
      `seuil Go = ${goValue}, miroir front = ${WEAPON_RANGE_MIN_MEASURED} : mettre à jour ` +
        'WEAPON_RANGE_MIN_MEASURED dans weaponRange_logic.ts (la phrase « sous le seuil de N ' +
        'mesures » ment tant que les deux divergent).',
    ).toBe(goValue)
  })
})

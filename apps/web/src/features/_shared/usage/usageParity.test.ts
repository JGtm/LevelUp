/**
 * usageParity.test.ts — la parité du dénominateur « mon camp / lobby » du bloc « usages
 * d'équipement, armes spéciales et objectifs » (S3). Extrait de `usageLogic.test.ts` le
 * 2026-09-09 (étape E5.1bis, scission de taille — CLAUDE.md n°5) au moment du déménagement
 * du bloc vers `features/_shared/usage/`.
 */
import { describe, expect, it } from 'vitest'

import { teamOfLobbyParityPct } from './usageParity'

describe('parité du dénominateur « mon camp / lobby »', () => {
  it('dérive 100 × équipe / lobby, et refuse un lobby vide ou absent', () => {
    expect(teamOfLobbyParityPct(4, 8)).toBe(50)
    expect(teamOfLobbyParityPct(undefined, 8)).toBeNull()
    expect(teamOfLobbyParityPct(4, 0)).toBeNull()
  })
})

/**
 * usageAvailability.test.ts — les états du bloc « usages d'équipement, armes spéciales et
 * objectifs » (S3). Extrait de `usageLogic.test.ts` le 2026-09-09 (étape E5.1bis, scission de
 * taille — CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/`.
 */
import { describe, expect, it } from 'vitest'

import type { SessionUsageBlock } from '@/lib/api/types'

import { usageAvailability } from './usageAvailability'
import { USAGE_TEXT } from './usageI18n'

const t = USAGE_TEXT.fr

describe('usageAvailability — états du bloc', () => {
  const base: SessionUsageBlock = {
    available: true,
    matches_measured: 8,
    matches_total: 9,
  }

  it('absent du payload → caché (pas de bloc fantôme)', () => {
    expect(usageAvailability(undefined, t)).toEqual({ kind: 'hidden' })
  })

  // LES DEUX RAISONS DE N'AVOIR RIEN NE SE DISENT PAS PAREIL (2026-09-05, registre L4).
  it("titre sans résumé d'usage (unsupported) → caché, pas de carte morte", () => {
    expect(
      usageAvailability({ ...base, available: false, unavailable_reason: 'unsupported' }, t),
    ).toEqual({ kind: 'hidden' })
  })

  it('lecture échouée (load_failed) → état vide AVEC la raison (transitoire, il faut le dire)', () => {
    expect(
      usageAvailability({ ...base, available: false, unavailable_reason: 'load_failed' }, t),
    ).toEqual({ kind: 'empty', message: t.unavailableLoadFailed })
  })

  it('0 match mesuré → état vide « aucun film », bloc présent', () => {
    expect(usageAvailability({ ...base, matches_measured: 0 }, t)).toEqual({
      kind: 'empty',
      message: t.unavailableNoMeasured,
    })
  })

  it('disponible et mesuré → ok', () => {
    expect(usageAvailability(base, t)).toEqual({ kind: 'ok' })
  })
})

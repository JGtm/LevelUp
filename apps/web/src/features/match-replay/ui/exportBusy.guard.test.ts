/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : la règle « l'export de la capture travaille-t-il ? » qui neutralise la lecture
 * vit dans `export/captureBusy.ts` (`isCaptureBusy`). La barre complète, la barre d'une ligne et le bouton
 * d'export la lisaient chacun à la main ; une quatrième copie re-divergerait en silence.
 */
import { describe, expect, it } from 'vitest'

import { cheminCourt, lire, nomDe, sourcesDeLaFeature } from '../test/featureFiles'

/** Les seuls fichiers qui nomment `isExportBusy` : sa définition, son usage interne, et la règle partagée. */
const AUTORISES = new Set(['ReplayExportDialog.tsx', 'captureBusy.ts'])

describe('garde-rail : la neutralisation pendant un export passe par isCaptureBusy', () => {
  it('isExportBusy ne se lit pas hors de la boîte d’export et de captureBusy.ts', () => {
    const coupables = sourcesDeLaFeature()
      .filter((f) => !AUTORISES.has(nomDe(f)) && /\bisExportBusy\b/.test(lire(f)))
      .map(cheminCourt)
    expect(coupables).toEqual([])
  })
})

/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6, 2026-09-16) : le chargement des images de vignettes passe par
 * `withLoadedImage` (`loadedImage.ts`). Quatre calques le recopiaient ; une copie qui rechargerait
 * l'image à chaque changement d'encre ferait sortir les premières images d'un export sans
 * vignettes (cf. l'en-tête de `loadedImage.ts`).
 *
 * UNE EXCEPTION NOMMÉE : `useReplayExport.ts` charge le filigrane d'équipe UNE fois par export,
 * sous promesse et avec gestion d'échec — un chargement ponctuel, pas une vignette reteinte.
 */
import { describe, expect, it } from 'vitest'

import { cheminCourt, lire, nomDe, sourcesDeLaFeature } from '../test/featureFiles'

const CHARGEMENT_DIRECT = /\bnew Image\(/
const AUTORISES = new Set(['loadedImage.ts', 'useReplayExport.ts'])

describe('garde-rail : les images des vignettes passent par withLoadedImage', () => {
  it('aucun `new Image(` ailleurs dans la feature (hors tests, hors helper, hors exception nommée)', () => {
    const fautifs = sourcesDeLaFeature()
      .filter((f) => !AUTORISES.has(nomDe(f)))
      .filter((f) => CHARGEMENT_DIRECT.test(lire(f)))
      .map(cheminCourt)
    expect(fautifs).toEqual([])
  })

  it('et le helper, lui, charge bien — sans quoi ce garde ne garderait rien', () => {
    const helper = sourcesDeLaFeature().find((f) => nomDe(f) === 'loadedImage.ts')
    expect(helper && CHARGEMENT_DIRECT.test(lire(helper))).toBe(true)
  })
})

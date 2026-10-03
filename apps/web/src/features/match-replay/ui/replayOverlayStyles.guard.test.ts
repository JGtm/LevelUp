/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : le style du panneau d'overlay du rejeu est CENTRALISÉ dans
 * `replayOverlayStyles.ts`, et le bloc de statut ne doit pas se re-écrire À LA MAIN
 * ailleurs dans la feature.
 *
 * POURQUOI. Le message inter-manche a été aligné sur « le même style que le texte de défaite ou
 * victoire » (retour utilisateur du 2026-08-28). C'était la 3e copie du même littéral de bloc
 * (panneau d'équipe + panneau neutre + message) : la règle impose de centraliser ET de poser ce
 * test, sans quoi une 4e copie re-divergerait en silence — et le style cesserait d'être « le
 * même ». Une re-inline du littéral casse CE test, pas le rendu.
 */
import { describe, expect, it } from 'vitest'
import { cheminCourt, lire, nomDe, sourcesDeLaFeature } from '../test/featureFiles'

import { OVERLAY_STATUS_BLOCK } from './replayOverlayStyles'

const STYLES_FILE = 'replayOverlayStyles.ts'

describe('garde-rail : le bloc de statut des overlays est centralisé', () => {
  it('le littéral exact du bloc ne vit QUE dans replayOverlayStyles.ts', () => {
    const coupables = sourcesDeLaFeature()
      .filter((f) => nomDe(f) !== STYLES_FILE && lire(f).includes(OVERLAY_STATUS_BLOCK))
      .map(cheminCourt)
    expect(coupables).toEqual([])
  })

  it('la constante de bloc est bien celle attendue (le style « victoire/défaite »)', () => {
    // Si ce littéral change, l'écran de fin ET le message inter-manche changent ENSEMBLE — c'est
    // exactement l'invariant que la centralisation garantit.
    expect(OVERLAY_STATUS_BLOCK).toBe(
      'rounded-lg px-8 py-4 text-center text-2xl font-bold uppercase tracking-wide text-foreground shadow-lg backdrop-blur-sm',
    )
  })
})

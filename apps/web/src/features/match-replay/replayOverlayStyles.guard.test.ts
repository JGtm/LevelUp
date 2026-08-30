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
import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { OVERLAY_STATUS_BLOCK } from './replayOverlayStyles'

const FEATURE_DIR = __dirname
const STYLES_FILE = 'replayOverlayStyles.ts'

/** Les fichiers de la feature où un littéral de classe pourrait se recopier (hors tests). */
function sourceFiles(): string[] {
  return readdirSync(FEATURE_DIR).filter(
    (f) => (f.endsWith('.ts') || f.endsWith('.tsx')) && !f.endsWith('.test.ts') && !f.endsWith('.test.tsx'),
  )
}

describe('garde-rail : le bloc de statut des overlays est centralisé', () => {
  it('le littéral exact du bloc ne vit QUE dans replayOverlayStyles.ts', () => {
    const coupables = sourceFiles().filter(
      (f) => f !== STYLES_FILE && readFileSync(resolve(FEATURE_DIR, f), 'utf8').includes(OVERLAY_STATUS_BLOCK),
    )
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

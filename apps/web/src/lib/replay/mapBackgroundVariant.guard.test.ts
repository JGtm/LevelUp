/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail — LES TROIS VUES 2D DÉCIDENT DU CALQUE DES MURS PAR LE MÊME PRÉDICAT.
 *
 * Le rejeu, la Tactique et « Occupation du terrain » retirent le plan des murs quand le fond posé est
 * l'image plan, qui le porte déjà. Trois copies de `show && variante === 'rendu'` auraient divergé :
 * `isWallPlanDrawn` (`mapBackgroundVariant.ts`) est la seule définition, et chaque vue l'appelle.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const SRC = resolve(__dirname, '../..')
const VUES = [
  'features/match-replay/ui/ReplayCanvas.tsx',
  'features/tactical/TacticalPlanCard.tsx',
  'features/match-view/MatchPositionsHeatmap.tsx',
]

describe('garde-rail : le calque des murs en mode plan', () => {
  it.each(VUES)('%s décide par isWallPlanDrawn', (vue) => {
    const source = readFileSync(resolve(SRC, vue), 'utf8')
    expect(source).toMatch(/isWallPlanDrawn\(/)
    expect(source).not.toMatch(/showWallPlan\s*&&/)
  })
})

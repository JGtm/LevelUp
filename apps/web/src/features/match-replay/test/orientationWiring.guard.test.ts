/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail — LE CANVAS DONNE L'ORIENTATION ET LE PLAN DES MURS À CEUX QUI EN ONT BESOIN.
 *
 * POURQUOI UN TEST DE TEXTE (même raison que `playerFocusWiring.guard`) : le canvas n'a pas de test
 * de rendu. Retirer l'une de ces liaisons laisserait tous les tests de calque verts :
 *  - sans `turns` au cadrage, la carte ne tourne plus, alors que la commande répond ;
 *  - sans `turns` à Suivre, la carte reste entière après une rotation jusqu'au prochain déplacement ;
 *  - un plan cuit quand son interrupteur est éteint recuit un calque invisible à chaque zoom.
 */
import { describe, expect, it } from 'vitest'

import { fichierNomme, lire } from './featureFiles'

const canvas = () => lire(fichierNomme('ReplayCanvas.tsx'))

describe('garde-rail : orientation et plan des murs câblés dans le canvas', () => {
  it('le cadrage prend l’orientation retenue pour la carte', () => {
    expect(canvas()).toContain('turns: orientation.turns')
    expect(canvas()).toMatch(/useReplayView\(\{[^}]*turns: orientation\.turns/)
  })

  it('Suivre recadre au changement d’orientation', () => {
    expect(canvas()).toMatch(/useReplayPlayerFocus\(\{[^}]*turns: orientation\.turns/)
  })

  it('le panneau de la vue reçoit l’orientation', () => {
    expect(canvas()).toMatch(/useReplayViewMenu\(\{\s*settings, orientation,/)
  })

  it('le plan des murs n’est cuit que quand son interrupteur est allumé et que le fond n’est pas déjà le plan', () => {
    expect(canvas()).toContain("const wallPlanOn = isWallPlanDrawn(showWallPlan, mapImage?.variante ?? 'rendu')")
    expect(canvas()).toContain('plan: wallPlanOn ? (mapImage?.plan ?? null) : null')
  })
})

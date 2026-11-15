/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : le couple « la souris ne prend pas le focus, Espace ne remonte pas
 * au raccourci de lecture » des boutons du rejeu vit dans `replayPressGuards.ts` et ne se réécrit
 * pas à la main. Quatre boutons le portent (ligne de frag, fiche de joueur, actions et retour du
 * panneau) ; une cinquième copie re-divergerait en silence.
 */
import { describe, expect, it } from 'vitest'

import { cheminCourt, lire, nomDe, sourcesDeLaFeature } from '../test/featureFiles'

const GUARDS_FILE = 'replayPressGuards.ts'

describe('garde-rail : les gardes de pression des boutons sont centralisés', () => {
  it('le mousedown annulé ne s’écrit pas à la main hors replayPressGuards.ts', () => {
    const coupables = sourcesDeLaFeature()
      .filter((f) => nomDe(f) !== GUARDS_FILE && /onMouseDown=\{\(e\) => e\.preventDefault\(\)\}/.test(lire(f)))
      .map(cheminCourt)
    expect(coupables).toEqual([])
  })

  it('l’Espace gardé localement ne s’écrit pas à la main dans un bouton (hors menus à navigation propre)', () => {
    const coupables = sourcesDeLaFeature()
      .filter((f) => nomDe(f) !== GUARDS_FILE && /if \(e\.key === ' '\) e\.stopPropagation\(\)/.test(lire(f)))
      .map(cheminCourt)
    expect(coupables).toEqual([])
  })
})

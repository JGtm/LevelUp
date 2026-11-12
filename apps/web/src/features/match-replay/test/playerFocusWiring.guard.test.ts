/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail — LE PANNEAU DE JOUEUR EST CÂBLÉ À LA CARTE, ET IL NE TOUCHE PAS AU POINT DE VUE
 * (lot 4 du plan « maquettes du rejeu », items 4.4 à 4.6).
 *
 * POURQUOI UN TEST DE TEXTE. Le canvas n'a pas de test de rendu : il est monté avec une toile, des
 * dizaines de hooks et un document. Retirer UNE des liaisons de `shownAtSlot` laisserait tous les
 * tests verts — Masquer cesserait simplement de retirer un des éléments du joueur (son « ! », ses
 * éclairs…) sans que rien ne le dise. Les tests de calque prouvent que chaque calque honore le
 * prédicat ; celui-ci prouve que le canvas le lui DONNE.
 *
 * ET LE POINT DE VUE (item 4.6) : Seul, Masquer et Suivre ne décident ni de l'encre de camp, ni du
 * roster, ni de la frise. Le panneau ne lit donc ni n'écrit `useReplayViewpoint` : un lien entre les
 * deux ferait changer la couleur de tout le rejeu quand on masque un joueur.
 */
import { describe, expect, it } from 'vitest'

import { fichierNomme, lire } from './featureFiles'

const canvas = () => lire(fichierNomme('ReplayCanvas.tsx'))

describe('garde-rail : le canvas donne le prédicat Seul / Masquer à chaque élément propre au joueur', () => {
  it('le panneau de joueur est branché, et la main reprend le cadrage', () => {
    const src = canvas()
    expect(src).toContain('useReplayPlayerFocus({ doc, focus: playerFocus, frameRef, zoom })')
    expect(src).toContain('onManualMove: onFollowStop')
  })

  it.each([
    ['pistes (pion, nom, cône, traînée)', /embarkedAtSlot: vehicles\.isEmbarkedAt, seatedAtSlot, shownAtSlot,/],
    ['marques de tir', /k, embarkedAtSlot: vehicles\.isEmbarkedAt, shownAtSlot,/],
    ['éclairs de bouche', /vehicleSizeOf: vehicles\.sizeOf, shownAtSlot,/],
    ['sillage du propulseur', /useReplayAbilityFx\(\{[^}]*shownAtSlot[^}]*\}\)/],
    ['vols de projectile (grenades, tirs)', /drawProjectilesLayer\([^;]*?shownProjectile,?\s*\)/],
    ['fins de vol des grenades', /useReplayGrenadeRest\(\{[^}]*shownAtSlot[^}]*\}\)/],
    ['mise en valeur d’une élimination', /useReplayKillFocus\(\{[^}]*shownAtSlot[^}]*\}\)/],
  ])('%s', (_nom, motif) => {
    expect(canvas()).toMatch(motif)
  })

  it('Suivre gèle la cuisson des calques statiques comme le glisser (pas de recuisson par image)', () => {
    expect(canvas()).toMatch(/useReplayStaticLayers\(\{[^]*?frozen: drag\.dragging,\s*follow: playerFocus\?\.follow \?\? false,/)
  })

  it('la scène se relie quand le prédicat change (dépendance de buildScene)', () => {
    expect(canvas()).toMatch(/campOfSlot, seatedAtSlot, shownAtSlot, shownProjectile,/)
  })
})

describe('garde-rail : les actions du panneau ne touchent pas au point de vue (4.6)', () => {
  const FICHIERS = [
    'playerPanelLogic.ts', 'followLogic.ts', 'usePlayerPanel.ts', 'useReplayPlayerFocus.ts',
    'useReplayFollow.ts', 'ReplayPlayerPanel.tsx', 'ReplayPlayerPanelList.tsx', 'ReplayFeedOrPanel.tsx',
  ]

  it.each(FICHIERS)('%s ne lit ni n’écrit le point de vue', (nom) => {
    const code = lire(fichierNomme(nom))
      .replace(/\/\*[\s\S]*?\*\//g, '')
      .replace(/^\s*\/\/.*$/gm, '')
    expect(code).not.toMatch(/useReplayViewpoint|replayViewpoint|\bviewpoint\b|\bis_me\b/)
  })
})

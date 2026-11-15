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

/**
 * appelDe rend le texte de l'appel `nom(...)` jusqu'à sa parenthèse fermante, commentaires retirés.
 * Elle lève quand l'appel n'existe pas : un calque retiré du canvas ne laisse pas le test vert.
 */
function appelDe(source: string, nom: string): string {
  const code = source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|\s)\/\/.*$/gm, '$1')
  const debut = code.search(new RegExp(`\\b${nom}\\(`))
  if (debut < 0) throw new Error(`appel introuvable : ${nom}`)
  let profondeur = 0
  for (let i = code.indexOf('(', debut); i < code.length; i++) {
    if (code[i] === '(') profondeur++
    else if (code[i] === ')' && --profondeur === 0) return code.slice(debut, i + 1)
  }
  throw new Error(`appel non fermé : ${nom}`)
}

describe('garde-rail : le canvas donne le prédicat Seul / Masquer à chaque élément propre au joueur', () => {
  it('le panneau de joueur est branché, et la main reprend le cadrage', () => {
    const src = canvas()
    expect(src).toContain('useReplayPlayerFocus({ doc, focus: playerFocus, frameRef, zoom, turns: orientation.turns })')
    expect(src).toContain('onManualMove: onFollowStop')
  })

  it.each([
    ['pistes (pion, nom, cône, traînée)', /embarkedAtSlot: vehicles\.isEmbarkedAt, seatedAtSlot, shownAtSlot,/],
    ['marques de tir', /k, embarkedAtSlot: vehicles\.isEmbarkedAt, shownAtSlot,/],
    ['éclairs de bouche', /vehicleSizeOf: vehicles\.sizeOf, shownAtSlot,/],
    ['vols de projectile (grenades, tirs)', /drawProjectilesLayer\([^;]*?shownProjectile,?\s*\)/],
  ])('%s', (_nom, motif) => {
    expect(canvas()).toMatch(motif)
  })

  // CHAQUE LIAISON EST LUE DANS LES BORNES DE SON APPEL (parenthèses appariées, commentaires retirés) : un motif
  // lâche trouverait le même texte dans l'appel d'un AUTRE calque, et retirer la liaison laisserait le test vert.
  it.each([
    ['sillage du propulseur', 'useReplayAbilityFx', /\bshownAtSlot\b/],
    ['câble du grappin et poussée du répulseur', 'useReplayAbilityFx', /\bfocusFilter: focusShown\b/],
    ['fins de vol des grenades', 'useReplayGrenadeRest', /\bshownAtSlot\b/],
    ['mise en valeur d’une élimination', 'useReplayKillFocus', /\bshownAtSlot\b/],
    ['effets de mort', 'drawKillFxLayer', /\bshown: focusShown\?\.killFx\b/],
    ['carte de chaleur', 'useReplayHeatmap', /\bshown: focusShown\b/],
    ['armes au sol', 'useReplayGroundWeapons', /\bshown: focusShown\?\.groundWeapon\b/],
    ['poses d’équipement et failles', 'useReplayPlacements', /\bfocusFilter: focusShown\b/],
    ['marques de révélation du capteur', 'drawEquipmentPlacementsLayer', /\bshown: focusShown\b/],
    ['véhicules', 'useReplayVehicles', /\bshown: focusShown\b/],
    ['drapeaux et ondes de capture', 'useReplayFlagCarries', /\bshownAuthor: focusShown\?\.author\b/],
    ['couronne du VIP', 'useReplayVipCrown', /\bshownCarrier: focusShown\?\.author\b/],
    ['crâne porté', 'useReplaySkullCarrier', /\bshownCarrier: focusShown\?\.author\b/],
    ['bombe portée', 'useReplayBombCarrier', /\bshownCarrier: focusShown\?\.author\b/],
    ['déflagration', 'useReplayBombBlast', /\bshownAuthor: focusShown\?\.author\b/],
    ['pulses d’objectif', 'drawObjectivePulses', /\bshownAuthor: focusShown\?\.author\b/],
  ])('%s', (_nom, fonction, motif) => {
    expect(appelDe(canvas(), fonction)).toMatch(motif)
  })

  it('appelDe ne rend que l’appel nommé : un texte voisin ne le satisfait pas', () => {
    const src = 'a({ x: 1 })\nb({ shown: focusShown, f: g(1) })\nc({ y: 2 }) // shown: focusShown'
    expect(appelDe(src, 'b')).toContain('shown: focusShown')
    expect(appelDe(src, 'a')).not.toContain('shown: focusShown')
    expect(appelDe(src, 'c')).not.toContain('shown: focusShown')
    expect(() => appelDe(src, 'absent')).toThrow()
  })

  it('Suivre gèle la cuisson des calques statiques comme le glisser (pas de recuisson par image)', () => {
    expect(canvas()).toMatch(/useReplayStaticLayers\(\{[^]*?frozen: drag\.dragging,\s*follow: playerFocus\?\.follow \?\? false,/)
  })

  it('la scène se relie quand le prédicat change (dépendance de buildScene)', () => {
    expect(canvas()).toMatch(/campOfSlot, seatedAtSlot, shownAtSlot, shownProjectile, focusShown,/)
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

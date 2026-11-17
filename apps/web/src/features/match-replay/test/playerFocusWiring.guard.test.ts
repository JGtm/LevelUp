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

/** Le canvas = le composant ET son moteur : le composant câble les hooks, le moteur lie les peintres. */
const canvas = () => lire(fichierNomme('ReplayCanvas.tsx')) + '\n' + lire(fichierNomme('replayRenderer.ts'))

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
    expect(src).toContain('useReplayPlayerFocus({ doc, focus: playerFocus, followPlayer: cameraFollow.player, frameRef, zoom, turns })')
    expect(src).toContain('onManualMove: onFollowStop')
  })

  // LA CAMÉRA NE DÉPEND PAS DE LA FICHE : la toile suit `cameraFollow.player`, que la page remplit depuis
  // `followXuid` (fiche ouverte ou fermée), jamais depuis `playerFocus` (la fiche ouverte). Lire le drapeau
  // `follow` du focus pour décider du suivi ferait de nouveau taire la caméra quand la fiche est fermée.
  it('la caméra suit le joueur de `cameraFollow`, pas le drapeau de la fiche, et la page la remplit', () => {
    const src = canvas()
    expect(src).not.toMatch(/playerFocus\??\.follow/)
    expect(src).toContain('cameraFollow.player')
    expect(lire(fichierNomme('ReplayStage.tsx'))).toContain('cameraFollow={view.playerPanel.camera}')
    expect(lire(fichierNomme('usePlayerPanel.ts'))).toContain('followedPlayerOf(state, players)')
  })

  it.each([
    ['pistes (pion, nom, cône, traînée)', /embarkedAtSlot: vehicles\.isEmbarkedAt, seatedAtSlot, shownAtSlot,/],
    ['marques de tir', /k, embarkedAtSlot: vehicles\.isEmbarkedAt, shownAtSlot,/],
    ['éclairs de bouche', /vehicleSizeOf: vehicles\.sizeOf, shownAtSlot,/],
    ['vols de projectile (grenades, tirs)', /drawProjectilesLayer\([^;]*?shown: shownProjectile,?\s*\},?\s*\)/],
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
    expect(canvas()).toMatch(/useReplayStaticLayers\(\{[^]*?frozen: drag\.dragging,\s*follow: cameraFollow\.player !== null,/)
  })

  it('la scène se relie quand le prédicat change (dépendance de la scène)', () => {
    expect(canvas()).toMatch(/campOfSlot, seatedAtSlot, shownAtSlot, shownProjectile, focusShown,/)
  })

  // LE PION CERCLÉ de la fiche ouverte : le canvas donne le repère au moteur, la scène se relie quand il change, et
  // le moteur le peint avec les pions (même peintre, par-dessus eux).
  it('le pion cerclé de la fiche ouverte est donné à la scène et peint avec les pions', () => {
    const src = canvas()
    expect(src).toMatch(/focus: { shownAtSlot, shownProjectile, shown: focusShown, openPlayer }/)
    expect(src).toMatch(/shownProjectile, focusShown, openPlayer,/)
    expect(src).toContain('if (openPlayer) drawOpenPlayerRing(ctx, openPlayer, view, fr, k)')
    expect(lire(fichierNomme('useReplayPlayerFocus.ts'))).toContain('useReplayOpenPlayer(doc, focus)')
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

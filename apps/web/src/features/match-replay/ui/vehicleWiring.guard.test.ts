/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : LE CANVAS CÂBLE LES VÉHICULES AUX CALQUES QUI EN DÉPENDENT.
 *
 * Trois liaisons de `ReplayCanvas.tsx` que ni le compilateur ni les tests des calques ne voient :
 * chaque paramètre est optionnel ou de type `string`, et le calque testé seul reçoit ce que son
 * test lui donne.
 *
 *  1. LA PORTE DES MARQUES DE TIR. `drawFireMarks` reçoit `embarkedAtSlot` pour taire la marque
 *     d'un tireur embarqué : son pion n'est pas dessiné, une marque posée sur lui flotterait sur
 *     le véhicule. La porte doit être LA MÊME que celle du calque des pions
 *     (`drawTracksLayer`) : deux prédicats divergents montreraient une marque sans pion, ou un
 *     pion sans marque.
 *  2. LA PORTE DES PIONS. `drawTracksLayer` reçoit le même `vehicles.isEmbarkedAt`.
 *  3. L'ENCRE DES ÉLÉMENTS DE CARTE. `useReplayVehicles` reçoit `mapElementInk: zoneInk.fill` :
 *     un élément de carte non jouable (tourelle sans occupant) se peint du gris des zones
 *     neutres, achromatique dans toutes les palettes, jamais d'une couleur d'équipe.
 */
import { describe, expect, it } from 'vitest'

import { fichierNomme, lire } from '../test/featureFiles'

const PORTE = 'vehicles.isEmbarkedAt'

/** Le texte d'un appel, de `ouverture` à la parenthèse qui le ferme (parenthèses équilibrées). */
function appel(src: string, ouverture: string): string {
  const debut = src.indexOf(ouverture)
  expect(debut, `l'appel ${ouverture} est introuvable dans le canvas`).toBeGreaterThan(0)
  let profondeur = 0
  for (let i = debut + ouverture.indexOf('('); i < src.length; i++) {
    if (src[i] === '(') profondeur++
    else if (src[i] === ')' && --profondeur === 0) return src.slice(debut, i + 1)
  }
  throw new Error(`l'appel ${ouverture} n'est pas fermé`)
}

/** La valeur passée à `cle` dans un appel (jusqu'à la virgule, l'accolade ou la fin de ligne). */
function valeur(corps: string, cle: string): string | undefined {
  return new RegExp(`\\b${cle}:\\s*([^,}\\n]+)`).exec(corps)?.[1]?.trim()
}

describe('ReplayCanvas — câblage des véhicules', () => {
  const src = lire(fichierNomme('ReplayCanvas.tsx'))

  it('le prédicat embarqué vient du hook des véhicules', () => {
    expect(src).toMatch(/const vehicles = useReplayVehicles\(/)
  })

  it('les marques de tir et les pions partagent LA MÊME porte embarquée', () => {
    const marques = valeur(appel(src, 'drawFireMarks('), 'embarkedAtSlot')
    const pions = valeur(appel(src, 'drawTracksLayer('), 'embarkedAtSlot')
    expect(marques, 'drawFireMarks doit recevoir embarkedAtSlot').toBe(PORTE)
    expect(pions, 'le calque des pions doit recevoir embarkedAtSlot').toBe(PORTE)
  })

  it("les éléments de carte prennent l'encre des zones neutres", () => {
    expect(valeur(appel(src, 'useReplayVehicles('), 'mapElementInk')).toBe('zoneInk.fill')
  })
})

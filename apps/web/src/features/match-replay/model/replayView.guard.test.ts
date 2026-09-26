/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6 : helper canonique + garde-rail le même jour) — LE CADRAGE A UN
 * SEUL TYPE, ET UNE SEULE FAÇON DE PROJETER.
 *
 * POURQUOI CE GARDE (registre 2026-09-05, K3). Le type du cadrage était redéclaré NEUF fois à
 * l'identique dans la feature, et le passage monde → canvas se réécrivait à chaque site en
 * dépaquetant ses quatre champs à la main — vingt-neuf fois, dans vingt-deux fichiers. Deux de
 * ces champs sont des nombres de même type (`width`, `height`) : les intervertir ne produit
 * aucune erreur de compilation, seulement un calque étiré, sur un seul écran, qu'il faut
 * remarquer à l'oeil.
 *
 * CE QUI RESTE PERMIS, ET POURQUOI. `replayLogic` DÉFINIT `worldToCanvas` et `canvasScale` :
 * il ne peut pas passer par `replayView`, qui l'importe — ce serait un cycle. Son unique site
 * (`layerOffset`) garde donc l'écriture longue, et c'est la seule exception.
 */
import { describe, expect, it } from 'vitest'
import { cheminCourt, fichierNomme, lire, nomDe, tousLesFichiers } from '../test/featureFiles'

/** La signature du défaut : les quatre champs du cadrage dépaquetés un par un. */
const DEPAQUETAGE = /view\.bounds,\s*view\.width,\s*view\.height,\s*view\.pad/

/** La signature de l'autre défaut : une seconde déclaration du type du cadrage. */
const DECLARATION = /(?:export )?interface CanvasView \{/

/** Le foyer, et le module qui définit les primitives (cf. l'en-tête). */
const AUTORISES = new Set(['replayView.ts', 'replayLogic.ts', 'replayView.guard.test.ts'])

function fichiers(): string[] {
  return tousLesFichiers().filter((f) => !AUTORISES.has(nomDe(f)))
}

function fautifs(motif: RegExp): string[] {
  return fichiers()
    .filter((f) => motif.test(lire(f)))
    .map(cheminCourt)
}

describe('garde-rail : un seul cadrage, une seule projection', () => {
  it('personne ne redéclare le type du cadrage', () => {
    expect(fautifs(DECLARATION)).toEqual([])
  })

  it('personne ne dépaquette les quatre champs du cadrage', () => {
    expect(fautifs(DEPAQUETAGE)).toEqual([])
  })

  it('et `replayView` porte bien les deux — sans quoi ce test ne garderait rien', () => {
    const src = lire(fichierNomme('replayView.ts'))
    expect(DECLARATION.test(src)).toBe(true)
    expect(DEPAQUETAGE.test(src)).toBe(true)
  })
})

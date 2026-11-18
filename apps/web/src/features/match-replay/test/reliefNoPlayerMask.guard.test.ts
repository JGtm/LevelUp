/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : LA COUPE PAR ÉTAGE NE MASQUE AUCUN JOUEUR DANS LE REJEU (maquette D.1, choix 9).
 *
 * Un joueur présent reste toujours affiché, quelle que soit la coupe : elle module le FOND (Relief, Plan, Rendu),
 * ses traits, rien d'autre. Seule « Occupation du terrain » masque les positions hors de l'étage choisi, et ce n'est
 * pas ce chemin. Le garde tient deux faits :
 *  - les modules de la coupe (`useReplayRelief`, `reliefLogic`, la commande d'étage) ne lisent ni piste, ni
 *    point, ni identité de joueur, ni filtre du panneau de joueur — ils n'ont donc aucun moyen de les filtrer ;
 *  - dans la toile, l'état du relief n'alimente que le fond, les traits et le panneau de la vue : jamais le
 *    filtre « Seul / Masquer » (`focusShown`, `shownAtSlot`) ni les calques de joueurs.
 */
import { describe, expect, it } from 'vitest'

import { fichierNomme, lire } from './featureFiles'

/** Ce qu'un module de la coupe ne doit jamais lire : de quoi masquer un joueur. */
const DE_QUOI_MASQUER = /tracks|shownAtSlot|focusShown|playerFocus|useReplayPlayerFocus|useSlotIdentity|xuid|roster/

const MODULES_DE_LA_COUPE = ['useReplayRelief.ts', 'reliefLogic.ts', 'ReplayFloorControls.tsx']

describe('garde-rail : la coupe par étage ne masque aucun joueur', () => {
  it.each(MODULES_DE_LA_COUPE)('%s ne lit rien qui permette de masquer un joueur', (nom) => {
    const code = lire(fichierNomme(nom))
      // Les commentaires disent la règle : seul le code compte.
      .replace(/\/\*[\s\S]*?\*\//g, '')
      .replace(/^\s*\/\/.*$/gm, '')
    expect(code).not.toMatch(DE_QUOI_MASQUER)
  })

  it('dans la toile, le relief n’alimente que le fond, les traits et le panneau de la vue', () => {
    const canvas = lire(fichierNomme('ReplayCanvas.tsx'))
    const usages = [...canvas.matchAll(/\brelief\b[.,\s]/g)].length
    expect(usages).toBeGreaterThan(0)
    // Les seules lectures de `relief.<champ>` : le fond composé et les traits à cuire.
    const champs = new Set([...canvas.matchAll(/\brelief\.(\w+)/g)].map((m) => m[1]))
    expect([...champs].sort()).toEqual(['background', 'linesLayer'])
  })

  it('le relief ne passe jamais par les filtres de joueur de la toile', () => {
    const canvas = lire(fichierNomme('ReplayCanvas.tsx'))
    expect(canvas).not.toMatch(/useReplayPlayerFocus\([^)]*relief/)
    expect(canvas).not.toMatch(/shownAtSlot[^\n]*relief|relief[^\n]*shownAtSlot/)
  })
})

/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) — UNE SEULE RÈGLE POUR « LE PORTEUR N'A PAS DE POSITION ».
 *
 * POURQUOI, ET IL A UNE DATE. Le 2026-09-11 (lot 6.7 phase B2, item 1), les TROIS calques qui
 * posent un objet d'objectif sur son porteur — crâne d'Oddball, bombe d'Assaut, drapeau de CTF —
 * répondaient chacun à leur façon à la même image muette : le crâne et la bombe DISPARAISSAIENT
 * (`if (!w) continue`), le drapeau se figeait à l'ANCRE DU SPAN, une position périmée affichée
 * avec l'habillage du porté. Trois réponses pour un seul fait mesuré (533 images côté drapeau,
 * 694 côté crâne). La règle est désormais écrite une fois — `carriedGlyphPlaceAt` — et ce garde
 * empêche qu'une quatrième s'écrive ailleurs (la couronne du VIP est la prochaine candidate).
 *
 * CE QU'IL DÉTECTE :
 *  1. l'ANCRE DU SPAN reservie comme position de dessin (le défaut du drapeau) ;
 *  2. le balayage arrière des positions du porteur, redéclaré hors du module.
 *
 * CE QU'IL NE PRÉTEND PAS : un calque qui écrirait la même règle avec d'autres noms de variables
 * passerait — aucun test grep ne remplace une revue. Il bloque la copie la plus probable, celle
 * qui part du code existant.
 */
import { describe, expect, it } from 'vitest'
import { cheminCourt, fichierNomme, lire, nomDe, tousLesFichiers } from '../test/featureFiles'

/** L'ancre du span reservie telle quelle comme position de dessin — le défaut C9 du drapeau. */
const ANCRE_PERIMEE = /return \{ x: now\.x, y: now\.y \}/

/** Le balayage arrière des positions du porteur, redéclaré. */
const BALAYAGE_ARRIERE = /for \(let f = frame - 1/

const AUTORISES = new Set(['carriedGlyphPlace.ts', 'carriedGlyphPlace.guard.test.ts'])

function fautifs(motif: RegExp): string[] {
  return tousLesFichiers()
    .filter((f) => !AUTORISES.has(nomDe(f)))
    .filter((f) => motif.test(lire(f)))
    .map(cheminCourt)
}

describe('garde-rail : une seule règle pour le glyphe porté sans position', () => {
  it('personne ne reserve l’ancre du span comme position de dessin', () => {
    expect(fautifs(ANCRE_PERIMEE)).toEqual([])
  })

  it('personne ne redéclare le balayage arrière des positions du porteur', () => {
    expect(fautifs(BALAYAGE_ARRIERE)).toEqual([])
  })

  it('et `carriedGlyphPlace` porte bien le balayage — sans quoi ce test ne garderait rien', () => {
    const src = lire(fichierNomme('carriedGlyphPlace.ts'))
    expect(BALAYAGE_ARRIERE.test(src)).toBe(true)
  })

  it('les TROIS consommateurs passent par la règle commune', () => {
    // Dérivé de la source, pas d'une liste écrite : tout fichier qui nomme la fonction la
    // consomme, et les trois calques de porteur DOIVENT y être.
    const consommateurs = tousLesFichiers()
      .filter((f) => !AUTORISES.has(nomDe(f)) && !nomDe(f).endsWith('.test.ts'))
      .filter((f) => /carriedGlyphPlaceAt/.test(lire(f)))
      .map(nomDe)
      .sort()
    // Le crâne passe par `skullPresence` (précédence portage/libre arbitrée là-bas), la bombe et
    // le drapeau appellent la règle depuis leur propre calque.
    expect(consommateurs).toEqual(['bombCarrierLayer.ts', 'flagCarriesLayer.ts', 'skullPresence.ts'])
  })
})

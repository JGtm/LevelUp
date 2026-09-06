/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6 : helper canonique + garde-rail le même jour) — L'ORIGINE DU
 * FILM NE SE LIT PLUS DANS L'ARTEFACT, ELLE SE DEMANDE À L'HORLOGE.
 *
 * POURQUOI CE GARDE. `doc.originMs` était lu à cinq endroits de la page de rejeu, et les
 * cinq répondaient différemment à son absence : `null` pour la fenêtre de gameplay, un
 * décalage MESURÉ par appariement pour le fil des éliminations, `0` pour les médias, la
 * présence et les sièges (registre 2026-09-05, P0-5). Sur un artefact sans origine — 5 des
 * 106 artefacts locaux — la même frise portait donc deux décalages : une capture et le frag
 * qu'elle montre pouvaient s'éloigner de l'écart réel entre les deux axes, mesuré de 3,6 s à
 * 50,8 s selon le match.
 *
 * La lecture a désormais UN foyer, `model/replayClock` (qui délègue les conversions à
 * `lib/replay/matchClock`), et un seul verdict : établie, ou pas. Ce test échoue si un
 * fichier du rejeu 2D — ou sa route — relit le champ par lui-même.
 *
 * CE QUI RESTE PERMIS : lire `originMs` SUR L'HORLOGE (`clock.originMs`), qui est justement
 * ce que ce garde veut voir se généraliser.
 */
import { describe, expect, it } from 'vitest'
import { resolve } from 'node:path'

import { cheminCourt, featureRoot, fichiersSous, lire, nomDe, tousLesFichiers } from '../test/featureFiles'

/**
 * La signature du défaut : une lecture de `originMs` sur autre chose qu'une horloge. La
 * négation arrière écarte `clock.originMs` / `replayClock.originMs`, la lecture légitime.
 */
const LECTURE_BRUTE = /(?<![Cc]lock\??)\.originMs/

/**
 * Les deux seuls fichiers du rejeu autorisés à nommer le champ : l'horloge, qui EST le
 * foyer, et la frontière de nullabilité du document, qui le recopie sans le lire. Ce garde
 * s'exempte lui-même — il porte la signature qu'il cherche.
 */
const AUTORISES = new Set(['replayClock.ts', 'replayNormalize.ts', 'replayClock.guard.test.ts'])

function fautif(fichier: string): boolean {
  if (AUTORISES.has(nomDe(fichier))) return false
  return LECTURE_BRUTE.test(lire(fichier))
}

describe('garde-rail : une seule lecture de l’origine du film', () => {
  it('aucun fichier de features/match-replay/ ne lit originMs hors de l’horloge', () => {
    expect(tousLesFichiers().filter(fautif).map(cheminCourt)).toEqual([])
  })

  it('la route du rejeu non plus', () => {
    const routes = resolve(featureRoot(), '..', '..', 'routes')
    expect(fichiersSous(routes).filter(fautif)).toEqual([])
  })

  it('et matchClock.ts, lui, la lit bien — sans quoi ce test ne garderait rien', () => {
    const foyer = resolve(featureRoot(), '..', '..', 'lib', 'replay', 'matchClock.ts')
    expect(LECTURE_BRUTE.test(lire(foyer))).toBe(true)
  })
})

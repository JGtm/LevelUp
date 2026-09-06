/// <reference types="node" />
// @vitest-environment node
/**
 * carrierPosition.guard.test.ts — LE GARDE-RAIL du chemin unique « position d'un porteur ».
 *
 * POURQUOI, ET IL A UNE DATE. Le 2026-09-05, la règle « joueur embarqué -> position du véhicule »
 * est entrée pour cinq calques d'un coup (bombe portée, couronne VIP, crâne d'Oddball, drapeau
 * porté, déflagration d'Assaut), puis pour DEUX lecteurs purs de plus le même jour, sur décision
 * du pilote : les effets de mort (`killFx.ts` — un joueur tué au volant explose sur son véhicule)
 * et les pulsations d'objectif (`objectivesLayer.ts`). Sept copies de la même règle, ce seraient
 * sept défauts qui divergent en silence — règle n° 6 du dépôt : on centralise ET on pose le
 * garde-rail, sans quoi la factorisation re-diverge (leçon du prédicat bot, 8 -> 36 copies).
 *
 * CE QU'IL DÉTECTE :
 *  1. un lecteur qui reprendrait la position de BIPÈDE seule (`buildPlayerPosAt`) au lieu du
 *     résolveur — le glyphe, l'effet ou le pulse retraverserait le décor en ligne droite ;
 *  2. une deuxième lecture de la position d'un véhicule (`vehiclePositionAt`) hors des trois
 *     écritures légitimes : sa définition, le tracé du sprite, et le résolveur.
 *
 * SA LISTE EST DÉRIVÉE DE LA SOURCE DEPUIS LE 2026-09-06 (registre, M4). Elle était ÉCRITE À
 * LA MAIN — cinq calques nommés un par un — et un SIXIÈME calque de porteur y échappait donc
 * par construction : il suffisait de l'oublier. La règle se lit maintenant dans le code
 * lui-même : tout module qui se donne un résolveur de position (`const posOf = …`) est un
 * lecteur de porteur, et doit le prendre du résolveur commun. Un calque neuf entre dans la
 * liste le jour où il écrit sa première ligne.
 *
 * CE QU'IL NE PRÉTEND PAS : un lecteur qui nommerait sa variable autrement passerait — aucun
 * test grep ne remplace une revue. Il bloque la copie la plus probable, celle qui part du code
 * existant, et il ne peut plus rater un calque par simple oubli.
 */
import { describe, expect, it } from 'vitest'

import { cheminCourt, lire, nomDe, sourcesDeLaFeature } from './test/featureFiles'

/**
 * LA SIGNATURE D'UN LECTEUR DE PORTEUR : il se donne un résolveur de position. C'est de là que
 * la liste se DÉRIVE — aucun nom de fichier n'est écrit ici.
 */
const RESOLVEUR = /const posOf = /

/** Tous les fichiers de la feature, hors tests (chemins absolus, toute l'arborescence). */
function sources(): string[] {
  return sourcesDeLaFeature()
}

/** Les lecteurs de porteur, dérivés : ceux qui se donnent un résolveur de position. */
function lecteurs(): string[] {
  return sources().filter((f) => RESOLVEUR.test(lire(f)))
}

/** Un lecteur monté dans React prend la version mémoïsée ; un lecteur pur, le constructeur. */
const EN_REACT = /^use[A-Z]/

/** Les seules écritures autorisées de la position d'un VÉHICULE (hors tests). */
const LECTEURS_POSITION_VEHICULE = new Set([
  'vehiclesLayer.ts', // la définition elle-même
  'vehiclesPaint.ts', // le sprite sur la carte
  'carrierPosition.ts', // la position d'un joueur embarqué
])

describe('garde-rail : un seul chemin pour la position d’un joueur embarqué', () => {
  it('la liste des lecteurs de porteur se dérive bien de la source, et n’est pas vide', () => {
    // Sept aujourd'hui : cinq hooks de calque et deux lecteurs purs. Le nombre n'est pas figé
    // — c'est la DÉRIVATION qui l'est : un calque neuf y entre tout seul.
    expect(lecteurs().length).toBeGreaterThanOrEqual(7)
  })

  it('tout lecteur de porteur passe par le résolveur commun', () => {
    const fautifs = lecteurs().filter((f) => {
      const src = lire(f)
      const attendu = EN_REACT.test(nomDe(f)) ? 'useCarrierPosAt(doc)' : 'buildCarrierPosAt(doc)'
      return !src.includes(attendu)
    })
    expect(
      fautifs,
      `ces lecteurs ne prennent plus la position au résolveur commun : [${fautifs.map(cheminCourt).join(', ')}]. ` +
        `Un porteur embarqué y retraverserait le décor en ligne droite (carrierPosition.ts).`,
    ).toEqual([])
  })

  it('aucun d’eux ne reprend la position de bipède seule, ni ne relit un véhicule lui-même', () => {
    const fautifs = lecteurs().filter((f) => {
      const src = lire(f)
      return src.includes('buildPlayerPosAt') || src.includes('vehiclePositionAt')
    })
    expect(
      fautifs,
      `la règle « embarqué -> véhicule » se recopie hors de carrierPosition.ts : ` +
        `[${fautifs.map(cheminCourt).join(', ')}].`,
    ).toEqual([])
  })

  it('la position d’un véhicule n’est lue que par ses trois écritures légitimes', () => {
    const fautifs = sources()
      .filter((f) => !LECTEURS_POSITION_VEHICULE.has(nomDe(f)))
      .filter((f) => lire(f).includes('vehiclePositionAt'))
    expect(
      fautifs,
      `nouvelle lecture de la position d'un véhicule hors des trois autorisées : ` +
        `[${fautifs.map(cheminCourt).join(', ')}].`,
    ).toEqual([])
  })
})

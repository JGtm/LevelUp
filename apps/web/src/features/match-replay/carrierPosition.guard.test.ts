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
 * CE QU'IL NE PRÉTEND PAS : un huitième lecteur qui naîtrait sans jamais nommer aucune de ces
 * fonctions passerait — aucun test grep ne remplace une revue. Il bloque la copie la plus
 * probable, celle qui part du code existant.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

/** Les cinq HOOKS de calque, qui prennent la version mémoïsée. */
const CALQUES_PORTEURS = [
  'useReplayBombCarrier.ts',
  'useReplayVipCrown.ts',
  'useReplaySkullCarrier.ts',
  'useReplayFlagCarries.ts',
  'useReplayBombBlast.ts',
]

/** Les deux lecteurs PURS, hors React, qui prennent le constructeur. */
const LECTEURS_PURS = ['killFx.ts', 'objectivesLayer.ts']

/** Les seules écritures autorisées de la position d'un VÉHICULE (hors tests). */
const LECTEURS_POSITION_VEHICULE = new Set([
  'vehiclesLayer.ts', // la définition elle-même
  'vehiclesPaint.ts', // le sprite sur la carte
  'carrierPosition.ts', // la position d'un joueur embarqué
])

function lire(fichier: string): string {
  return readFileSync(join(__dirname, fichier), 'utf8')
}

describe('garde-rail : un seul chemin pour la position d’un joueur embarqué', () => {
  it('les cinq calques de porteur passent par useCarrierPosAt', () => {
    const fautifs = CALQUES_PORTEURS.filter((f) => !lire(f).includes('useCarrierPosAt(doc)'))
    expect(
      fautifs,
      `ces calques ne lisent plus la position par le résolveur commun : [${fautifs.join(', ')}]. ` +
        `Un porteur embarqué y retraverserait le décor en ligne droite (carrierPosition.ts).`,
    ).toEqual([])
  })

  it('les deux lecteurs purs passent par buildCarrierPosAt', () => {
    const fautifs = LECTEURS_PURS.filter((f) => !lire(f).includes('buildCarrierPosAt(doc)'))
    expect(
      fautifs,
      `ces lecteurs ne passent plus par le résolveur commun : [${fautifs.join(', ')}]. ` +
        `Un effet de mort ou un pulse d'objectif s'y poserait là où personne n'était.`,
    ).toEqual([])
  })

  it('aucun d’eux ne reprend la position de bipède seule, ni ne relit un véhicule lui-même', () => {
    const fautifs = [...CALQUES_PORTEURS, ...LECTEURS_PURS].filter((f) => {
      const src = lire(f)
      return src.includes('buildPlayerPosAt') || src.includes('vehiclePositionAt')
    })
    expect(
      fautifs,
      `la règle « embarqué -> véhicule » se recopie hors de carrierPosition.ts : ` +
        `[${fautifs.join(', ')}].`,
    ).toEqual([])
  })

  it('la position d’un véhicule n’est lue que par ses trois écritures légitimes', () => {
    const fautifs = readdirSync(__dirname)
      .filter((f) => /\.tsx?$/.test(f) && !/\.test\.tsx?$/.test(f))
      .filter((f) => !LECTEURS_POSITION_VEHICULE.has(f))
      .filter((f) => lire(f).includes('vehiclePositionAt'))
    expect(
      fautifs,
      `nouvelle lecture de la position d'un véhicule hors des trois autorisées : ` +
        `[${fautifs.join(', ')}].`,
    ).toEqual([])
  })
})

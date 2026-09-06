/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : CHAQUE ID DE CALQUE EST LIÉ AU PEINTRE DE MÊME NOM.
 *
 * POURQUOI (2026-09-06, revue R1 du lot v2 D, constat C2). `buildScene` associe 25 ids à
 * 25 peintres. Les 25 valeurs ont le même type (`LayerPaint`) : le compilateur ne peut pas
 * départager. La revue a interverti `'couronne-vip': skullCarrier.paint` et
 * `'crane-porte': vipCrown.paint` — 2 350 tests verts, `tsc` exit 0, et à l'écran la couronne
 * du VIP remplacée par le crâne d'Oddball. C'est la seule faute que la refonte D.7 pouvait
 * introduire : elle a mis l'ORDRE et les CONDITIONS sous test, jamais la LIAISON.
 *
 * DEUX MOITIÉS, PARCE QUE LES CALQUES SE CÂBLENT DE DEUX FAÇONS.
 *  - ONZE calques sont câblés par un hook : depuis ce lot, le hook porte son `id` à côté de
 *    son `paint` et `bindPainters` en dérive la liaison. La faute n'est plus écrivable — ce
 *    fichier vérifie que la dérivation MARCHE (oracle sur `bindPainters`) et que le canvas
 *    ne renomme aucun d'eux à la main.
 *  - QUATORZE calques sont des fermetures écrites dans la table. Leur peintre nomme une
 *    fonction de dessin (ou une cuisson) : la table attendue est écrite À LA MAIN ci-dessous
 *    et confrontée à la source. Un swap y déplace le nom de la fonction appelée.
 */
import { describe, expect, it } from 'vitest'

import { bindPainters, LAYER_ORDER, type LayerPaint, type ReplayLayerId } from './replayCompose'
import { fichierNomme, lire } from '../test/featureFiles'

/** Les onze calques dont le hook porte l'identité (id -> hook qui le câble). */
const CABLES_PAR_HOOK: ReadonlyArray<[ReplayLayerId, string]> = [
  ['socles-armes', 'useReplayWeaponPads.ts'],
  ['armes-au-sol', 'useReplayGroundWeapons.ts'],
  ['vehicules', 'useReplayVehicles.ts'],
  ['gestes-capacite', 'useReplayAbilityFx.ts'],
  ['fin-de-vol', 'useReplayGrenadeRest.ts'],
  ['drapeaux', 'useReplayFlagCarries.ts'],
  ['objets-objectif', 'useReplayObjectiveObjects.ts'],
  ['couronne-vip', 'useReplayVipCrown.ts'],
  ['crane-porte', 'useReplaySkullCarrier.ts'],
  ['bombe-portee', 'useReplayBombCarrier.ts'],
  ['deflagration', 'useReplayBombBlast.ts'],
]

/**
 * Les quatorze fermetures de la table, et CE QU'ELLES DOIVENT PEINDRE. Oracle écrit à la
 * main : chaque id est suivi du symbole que son peintre doit nommer.
 */
const FERMETURES: ReadonlyArray<[ReplayLayerId, string]> = [
  ['fond-carte', 'ctx.drawImage(mapImage.image'],
  ['sol-forge', 'drawGeometryLayer('],
  ['chaleur', 'cuit(heatRef.current)'],
  ['zones-nommees', 'cuit(zonesRef.current)'],
  ['objectifs-cuits', 'cuit(objectivesRef.current)'],
  ['projectiles', 'drawProjectilesLayer('],
  ['poses-equipement', 'drawEquipmentPlacementsLayer('],
  ['trajectoires', 'drawTracksLayer('],
  ['marques-de-tir', 'drawFireMarks('],
  ['tirs', 'drawShotsLayer('],
  ['grenades', 'drawGrenadesLayer('],
  ['etat-zones', 'drawZoneStates('],
  ['pulses-objectif', 'drawObjectivePulses('],
  ['morts', 'drawKillFxLayer('],
]

/**
 * La seule TABLE DE LIAISON du canvas — le bloc `paint: {` de `buildScene`. Le découpage
 * commence là et pas au fichier entier : le bloc `has:` juste au-dessus porte les mêmes clés
 * (`projectiles`, `grenades`…) pour des booléens, et les confondre ferait lire une condition
 * à la place d'un peintre.
 */
function tableDeLiaison(): string {
  const src = lire(fichierNomme('ReplayCanvas.tsx'))
  const debut = src.indexOf('        paint: {')
  expect(debut, 'la table de liaison est introuvable dans le canvas').toBeGreaterThan(0)
  return src.slice(debut)
}

/** Le corps de l'entrée `id` de la table, jusqu'à l'entrée suivante. */
function entreeDeTable(table: string, id: ReplayLayerId): string {
  const cle = id.includes('-') ? `'${id}':` : `${id}:`
  const debut = table.indexOf(`\n          ${cle}`)
  expect(debut, `l'entrée ${id} est absente de la table de liaison`).toBeGreaterThan(0)
  const suite = table.slice(debut + 1)
  const fin = suite.search(/\n {10}(?:'[a-z-]+'|[a-z]+):/)
  return fin > 0 ? suite.slice(0, fin) : suite.slice(0, 600)
}
describe('garde-rail : la liaison id -> peintre', () => {
  it('les 25 ids de LAYER_ORDER sont couverts, une fois chacun', () => {
    const couverts = [...CABLES_PAR_HOOK.map(([id]) => id), ...FERMETURES.map(([id]) => id)]
    expect(couverts).toHaveLength(LAYER_ORDER.length)
    expect([...couverts].sort()).toEqual([...LAYER_ORDER].sort())
  })

  it('bindPainters prend l’id SUR le calque, jamais à côté', () => {
    const couronne: LayerPaint = () => {}
    const crane: LayerPaint = () => {}
    const lie = bindPainters(
      { id: 'couronne-vip', paint: couronne },
      { id: 'crane-porte', paint: crane },
    )
    expect(lie['couronne-vip']).toBe(couronne)
    expect(lie['crane-porte']).toBe(crane)
  })

  it('chacun des onze hooks de calque déclare SON id, et le rend', () => {
    for (const [id, hook] of CABLES_PAR_HOOK) {
      const src = lire(fichierNomme(hook))
      expect(src, `${hook} ne déclare pas l'id ${id}`).toContain(`id: '${id}'`)
      expect(src, `${hook} ne rend pas son id`).toContain(`return { id: '${id}',`)
    }
  })

  it('le canvas ne renomme AUCUN des onze : il dérive la liaison', () => {
    const table = tableDeLiaison()
    for (const [id] of CABLES_PAR_HOOK) {
      // La clé telle qu'elle s'écrirait dans la table, à son indentation exacte. Pas de
      // RegExp construite : dans un gabarit JS, `\s` vaut `s` — le motif serait inerte.
      const cle = id.includes('-') ? "'" + id + "':" : id + ':'
      expect(
        table.indexOf('\n          ' + cle),
        `le canvas réécrit à la main la liaison de ${id} : elle doit venir du hook`,
      ).toBe(-1)
    }
    expect(table).toContain('...bindPainters(')
  })

  it('chacune des quatorze fermetures peint bien ce que son id annonce', () => {
    const table = tableDeLiaison()
    for (const [id, attendu] of FERMETURES) {
      expect(entreeDeTable(table, id), `l'entrée ${id} ne peint pas ${attendu}`).toContain(attendu)
    }
  })
})

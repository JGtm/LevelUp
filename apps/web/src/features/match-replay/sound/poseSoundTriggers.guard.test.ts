/**
 * Garde-rail — CHAQUE SON DE POSE A UN DÉCLENCHEUR RÉEL DANS LES FIXTURES.
 *
 * Un son de pose qui ne part jamais est un son MORT, et un test qui lui fabrique une pose le cache :
 * c'est ce qui a tenu la dépose de la faille muette de début septembre au 14/10 (une pose `deployed`
 * de la balise, que le film ne produit plus depuis la règle du 15/09, fabriquée par
 * `replaySound.test.ts`). Ici, chaque famille de `EQUIPMENT_PLACEMENT_SOUND_STEMS` doit avoir au
 * moins une pose montrable (`placementIsDeployedObject`) dans les documents que Go produit
 * (`test/fixtures/go`).
 *
 * LES FAMILLES CI-DESSOUS SONT MORTES AU MÊME TITRE, ET CE N'EST PAS RÉGLÉ ICI (découverte du lot
 * « équipement porté », 2026-10-14) : au parc local (238 artefacts), seule la famille `wall` porte une
 * pose `deployed`. Le capteur, le traqueur, le champ de réparation et l'écran occultant ne sonnent
 * donc plus depuis le 15/09. Leur geste est désormais daté par le film (`carriedEquipment[].uses`) ;
 * leur rebranchement est une décision produit à prendre. Retrait de cette liste : dans le commit qui
 * rebranche chaque famille. Critère : la liste est vide.
 */
import { describe, expect, it } from 'vitest'

import { placementIsDeployedObject } from '../layers/equipmentPlacementsLayer'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { EQUIPMENT_PLACEMENT_SOUND_STEMS } from './replaySound'

const DECLENCHEURS_MORTS_CONNUS: ReadonlySet<string> = new Set([
  'sensor',
  'threat_seeker',
  'repair_field',
  'shroud_screen',
])

describe('sons de pose — un déclencheur réel', () => {
  it('chaque famille sonore a au moins une pose montrable dans les fixtures Go', () => {
    const vues = new Set<string>()
    for (const entry of goFixtureEntries()) {
      for (const p of loadGoFixture(entry).doc.equipmentPlacements ?? []) {
        if (placementIsDeployedObject(p)) vues.add(p.family)
      }
    }
    const muettes = Object.keys(EQUIPMENT_PLACEMENT_SOUND_STEMS).filter((f) => !vues.has(f))
    expect(muettes.filter((f) => !DECLENCHEURS_MORTS_CONNUS.has(f))).toEqual([])
    // Une famille de la liste qui retrouve un déclencheur en sort : la liste ne cache rien.
    expect([...DECLENCHEURS_MORTS_CONNUS].filter((f) => vues.has(f))).toEqual([])
  })
})

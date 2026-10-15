/**
 * Garde-rail — CHAQUE SON DE POSE, D'UTILISATION OU DE FIN D'ÉQUIPEMENT A UN DÉCLENCHEUR RÉEL.
 *
 * Un son qui ne part jamais est un son MORT, et un test qui lui fabrique une pose le cache : c'est ce
 * qui a tenu la dépose de la faille muette de début septembre au 14/10 (une pose `deployed` de la
 * balise, que le film ne produit plus depuis la règle du 15/09, fabriquée par `replaySound.test.ts`),
 * et le capteur, le traqueur, le champ de réparation et l'écran occultant muets du 15/09 au 15/10.
 *
 * Chaque stem de ces quatre tables doit avoir au moins un déclencheur dans des documents RÉELS :
 *  - `EQUIPMENT_PLACEMENT_SOUND_STEMS` : une pose montrable (`placementIsDeployedObject`) ;
 *  - `CARRIED_USE_SOUND_STEMS` : une utilisation (`carriedEquipment[].uses`) d'un objet de la famille ;
 *  - `CARRIED_END_SOUND_STEMS` : une fin lue d'un objet engendré (`carriedEquipment[].tracked[].t1`) ;
 *  - `RIFT_POSE_SOUND_STEM` : une pose de faille (`riftPoses`).
 * Les documents sont ceux que Go produit (`test/fixtures/go`), et, pour une famille qu'aucun des huit
 * builds ne porte (le traqueur), les relevés réels figés de `test/carriedUseWitnesses.ts`. AUCUNE
 * EXCEPTION : une famille sans déclencheur fait rougir ce test.
 *
 * Une famille n'est sonnée que par UNE des deux tables (pose ou utilisation) : le mur, dont la pose
 * suit l'utilisation de deux ou trois images, sonnerait deux fois.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { placementIsDeployedObject } from '../layers/equipmentPlacementsLayer'
import { documentsTemoinsDeLEquipementPorte } from '../test/carriedUseWitnesses'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { testReplayDoc } from '../test/testDoc'
import { CARRIED_END_SOUND_STEMS, CARRIED_USE_SOUND_STEMS } from './carriedUseSound'
import { EQUIPMENT_PLACEMENT_SOUND_STEMS } from './replaySound'
import { RIFT_POSE_SOUND_STEM } from './riftPoseSound'

function documentsGo(): ReplayDocumentReady[] {
  return goFixtureEntries().map((entry) => testReplayDoc(loadGoFixture(entry).doc))
}

function famillesPosees(docs: ReplayDocumentReady[]): Set<string> {
  const vues = new Set<string>()
  for (const doc of docs) {
    for (const p of doc.equipmentPlacements) if (placementIsDeployedObject(p)) vues.add(p.family)
  }
  return vues
}

function famillesUtilisees(docs: ReplayDocumentReady[]): Set<string> {
  const vues = new Set<string>()
  for (const doc of docs) {
    for (const c of doc.carriedEquipment) {
      const family = doc.abilityLabels?.[String(c.r)]?.family
      if (family && c.uses.length > 0) vues.add(family)
    }
  }
  return vues
}

function famillesFinies(docs: ReplayDocumentReady[]): Set<string> {
  const vues = new Set<string>()
  for (const doc of docs) {
    for (const c of doc.carriedEquipment) {
      const family = doc.abilityLabels?.[String(c.r)]?.family
      if (family && c.tracked.some((s) => s.t1 !== undefined)) vues.add(family)
    }
  }
  return vues
}

describe('sons d équipement — un déclencheur réel, sans exception', () => {
  it('chaque famille de pose a une pose montrable dans les fixtures Go', () => {
    const vues = famillesPosees(documentsGo())
    expect(Object.keys(EQUIPMENT_PLACEMENT_SOUND_STEMS).filter((f) => !vues.has(f))).toEqual([])
  })

  it('chaque famille d utilisation a une utilisation dans les fixtures Go ou les relevés figés', () => {
    const vues = famillesUtilisees([...documentsGo(), ...documentsTemoinsDeLEquipementPorte()])
    expect(Object.keys(CARRIED_USE_SOUND_STEMS).filter((f) => !vues.has(f))).toEqual([])
  })

  it('chaque famille de fin a une fin lue dans les fixtures Go ou les relevés figés', () => {
    const vues = famillesFinies([...documentsGo(), ...documentsTemoinsDeLEquipementPorte()])
    expect(Object.keys(CARRIED_END_SOUND_STEMS).filter((f) => !vues.has(f))).toEqual([])
  })

  it(`la pose de la faille (${RIFT_POSE_SOUND_STEM}) a un déclencheur dans les fixtures Go`, () => {
    expect(documentsGo().some((doc) => doc.riftPoses.length > 0)).toBe(true)
  })

  it('aucune famille n est sonnée à la fois par sa pose et par son utilisation', () => {
    const pose = new Set(Object.keys(EQUIPMENT_PLACEMENT_SOUND_STEMS))
    expect(Object.keys(CARRIED_USE_SOUND_STEMS).filter((f) => pose.has(f))).toEqual([])
  })
})

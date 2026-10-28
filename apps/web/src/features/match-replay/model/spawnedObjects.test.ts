import { describe, expect, it } from 'vitest'

import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'

import {
  isPlacementActive,
  placementKind,
  PLACEMENT_RENDER,
  type PlacementToggles,
} from '../layers/equipmentPlacementsLayer'
import { project, SHROUD_DURATION_MS } from '../layers/placementShapes'
import { VIEW } from '../test/placementFixtures'
import { testReplayDoc } from '../test/testDoc'
import { zonePresenceAt } from './equipmentZones'
import { placementAt } from './placementHitTest'
import { sensorReveals } from './threatSensor'
import {
  drawablePlacements,
  SPAWNED_OBJECT_FAMILIES,
  SPAWNED_OBJECT_ID,
  SPAWNED_OBJECT_OWN_CHANNEL_FAMILIES,
  spawnedObjectDeployments,
  spawnedObjectPlacements,
} from './spawnedObjects'

const TOGGLES: PlacementToggles = { showUnnamed: false, showDropped: true }
const TIME = { frameMs: 100, frames: 1000 }

const LABELS = {
  '1': { en: 'Threat Sensor', fr: 'capteur', family: 'sensor' },
  '2': { en: 'Drop Wall', fr: 'mur', family: 'wall' },
  '3': { en: 'Threat Seeker', fr: 'traqueur', family: 'threat_seeker' },
  '4': { en: 'Shroud Screen', fr: 'ecran', family: 'shroud_screen' },
  '5': { en: 'Repair Field', fr: 'champ', family: 'repair_field' },
  '6': { en: 'Quantum Translocator', fr: 'translocateur', family: 'translocator_beacon' },
  '7': { en: 'Grapple', fr: 'grappin' },
}

/** Un objet engendré situé : l'image d'apparition `t0`, la fin lue `t1`, le repos `land` et sa position. */
function tracked(t0: number, land: number, over: Record<string, number | undefined> = {}) {
  return { t0, t1: land + 20, land, x: 3, y: 4, z: 1, ...over }
}

function docWith(r: number, objects: unknown[], extra: Record<string, unknown> = {}) {
  return testReplayDoc({
    frameCount: 1000,
    frameIntervalMs: 100,
    abilityLabels: LABELS,
    carriedEquipment: [{ slot: 540, r, t0: 10, t1: 900, tracked: objects }],
    ...extra,
  } as never)
}

describe('spawnedObjectPlacements — la pose fabriquée depuis un objet engendré situé', () => {
  it('fabrique, pour un capteur situé, une pose déployée du porteur, ancrée au repos', () => {
    const doc = docWith(1, [tracked(100, 108, { x: 12.5, y: -3.25, z: 2 })])
    expect(spawnedObjectPlacements(doc)).toEqual([
      {
        family: 'sensor',
        id: SPAWNED_OBJECT_ID,
        origin: 'deployed',
        owner: 540,
        t0: 108,
        t1: 128,
        x: 12.5,
        y: -3.25,
        z: 2,
        endRead: true,
      },
    ])
  })

  it('couvre exactement les quatre familles nommées', () => {
    for (const [rank, family] of [
      [1, 'sensor'],
      [3, 'threat_seeker'],
      [4, 'shroud_screen'],
      [5, 'repair_field'],
    ] as const) {
      const poses = spawnedObjectPlacements(docWith(rank, [tracked(100, 105)]))
      expect(poses.map((p) => p.family)).toEqual([family])
    }
    expect([...SPAWNED_OBJECT_FAMILIES].sort()).toEqual(
      ['repair_field', 'sensor', 'shroud_screen', 'threat_seeker'],
    )
  })

  it('exclut le mur et la faille, même situés : ils ont leur propre canal', () => {
    expect(spawnedObjectPlacements(docWith(2, [tracked(100, 105)]))).toEqual([])
    expect(spawnedObjectPlacements(docWith(6, [tracked(100, 105)]))).toEqual([])
    for (const f of SPAWNED_OBJECT_OWN_CHANNEL_FAMILIES) {
      expect((SPAWNED_OBJECT_FAMILIES as readonly string[]).includes(f)).toBe(false)
    }
  })

  it("ne fabrique rien d'un objet non situé, ni d'un objet à position partielle", () => {
    const doc = docWith(1, [
      { t0: 100, t1: 120 },
      { t0: 100, t1: 120, land: 105 },
      { t0: 100, t1: 120, land: 105, x: 1, y: 2 },
      { t0: 100, t1: 120, x: 1, y: 2, z: 3 },
    ])
    expect(spawnedObjectPlacements(doc)).toEqual([])
  })

  it('ne fabrique rien pour un rang sans famille ou sans libellé', () => {
    expect(spawnedObjectPlacements(docWith(7, [tracked(100, 105)]))).toEqual([])
    expect(spawnedObjectPlacements(docWith(99, [tracked(100, 105)]))).toEqual([])
  })

  it('une fin lue antérieure au repos est ramenée au repos : la pose existe une image', () => {
    const [p] = spawnedObjectPlacements(docWith(1, [tracked(100, 105, { t1: 104 })]))
    expect(p.t0).toBe(105)
    expect(p.t1).toBe(105)
    expect(p.endRead).toBe(true)
  })

  it("sans fin lue la pose n'affirme aucune fin : ni t1 inventé, ni endRead", () => {
    const [p] = spawnedObjectPlacements(docWith(1, [tracked(100, 105, { t1: undefined })]))
    expect(p.t1).toBe(105)
    expect(p.endRead).toBeUndefined()
  })

  it("le traqueur et l'écran ne lisent jamais leur t1 : c'est la fin du projectile, pas de l'effet", () => {
    for (const rank of [3, 4]) {
      const [p] = spawnedObjectPlacements(docWith(rank, [tracked(100, 105, { t1: 107 })]))
      expect(p.endRead).toBeUndefined()
      expect(p.t0).toBe(105)
    }
  })
})

describe('fenêtre de dessin par famille (placementEndFrame via isPlacementActive)', () => {
  const active = (family: string, over: Record<string, number | undefined>, frame: number) => {
    const rank = { sensor: 1, threat_seeker: 3, shroud_screen: 4, repair_field: 5 }[family]!
    const [p] = spawnedObjectPlacements(docWith(rank, [tracked(100, 105, over)]))
    const kind = placementKind(p, TOGGLES)
    return kind !== null && isPlacementActive(p, kind, frame, TIME)
  }

  it('capteur à fin lue : de land à t1 inclus, sans prolongation par la durée officielle', () => {
    expect(active('sensor', { t1: 205 }, 104)).toBe(false)
    expect(active('sensor', { t1: 205 }, 105)).toBe(true)
    expect(active('sensor', { t1: 205 }, 205)).toBe(true)
    expect(active('sensor', { t1: 205 }, 206)).toBe(false)
    expect(active('sensor', { t1: 205 }, 400)).toBe(false)
  })

  it('champ à fin lue : de land à t1 inclus', () => {
    expect(active('repair_field', { t1: 150 }, 150)).toBe(true)
    expect(active('repair_field', { t1: 150 }, 151)).toBe(false)
  })

  it("capteur sans fin lue : la durée officielle de 15 s existante tient", () => {
    expect(active('sensor', { t1: undefined }, 105 + 150)).toBe(true)
    expect(active('sensor', { t1: undefined }, 105 + 151)).toBe(false)
  })

  it("champ sans fin lue : la règle existante de la famille (jusqu'à la fin du rejeu)", () => {
    expect(active('repair_field', { t1: undefined }, 999)).toBe(true)
  })

  it('traqueur : ancré au repos, fenêtre existante (la brièveté est celle de son impulsion)', () => {
    expect(active('threat_seeker', {}, 104)).toBe(false)
    expect(active('threat_seeker', {}, 105)).toBe(true)
  })

  it("écran : ancré au repos, fin = règle existante isolée dans SHROUD_DURATION_MS", () => {
    expect(active('shroud_screen', {}, 104)).toBe(false)
    expect(active('shroud_screen', {}, 105)).toBe(true)
    // Tant que l'utilisateur n'a pas tranché la durée, l'écran reste jusqu'à la fin du rejeu.
    expect(SHROUD_DURATION_MS).toBeUndefined()
    expect(active('shroud_screen', {}, 999)).toBe(true)
  })
})

describe('drawablePlacements — le canal de dessin', () => {
  const wall = {
    family: 'wall',
    id: '0x686b40c9',
    origin: 'deployed',
    owner: 565,
    t0: 1520,
    t1: 1524,
    x: 1,
    y: 2,
  }

  it('garde les poses du document en tête, dans leur ordre, puis ajoute les poses fabriquées', () => {
    const doc = docWith(1, [tracked(100, 105), tracked(200, 210)], { equipmentPlacements: [wall] })
    const all = drawablePlacements(doc)
    expect(all.slice(0, 1)).toEqual([wall])
    expect(all).toHaveLength(3)
    expect(all.slice(1).map((p) => p.t0)).toEqual([105, 210])
  })

  it("rend le tableau du document tel quel quand il n'y a rien à fabriquer", () => {
    const doc = docWith(2, [tracked(100, 105)], { equipmentPlacements: [wall] })
    expect(drawablePlacements(doc)).toBe(doc.equipmentPlacements)
  })

  it('ne modifie jamais equipmentPlacements du document (les sons le lisent)', () => {
    const doc = docWith(1, [tracked(100, 105)], { equipmentPlacements: [wall] })
    drawablePlacements(doc)
    expect(doc.equipmentPlacements).toEqual([wall])
  })

  it('est mémoïsé par document', () => {
    const doc = docWith(1, [tracked(100, 105)])
    expect(drawablePlacements(doc)).toBe(drawablePlacements(doc))
  })

  it('les poses fabriquées passent la porte du calque : la règle de rendu de leur famille', () => {
    const doc = docWith(1, [tracked(100, 105)])
    const [p] = spawnedObjectPlacements(doc)
    expect(placementKind(p, TOGGLES)).toBe(PLACEMENT_RENDER.sensor)
  })
})

describe('spawnedObjectDeployments — un objet engendré = un déploiement de sa famille', () => {
  it('compte aussi les objets sans position : le geste a eu lieu, seul son lieu manque', () => {
    const doc = docWith(1, [tracked(100, 105), { t0: 300, t1: 320 }])
    expect(spawnedObjectDeployments(doc)).toEqual([
      { family: 'sensor', owner: 540, t0: 100 },
      { family: 'sensor', owner: 540, t0: 300 },
    ])
  })

  it('exclut le mur et la faille : le mur se compte sur ses poses, la faille sur ses poses de faille', () => {
    expect(spawnedObjectDeployments(docWith(2, [tracked(100, 105)]))).toEqual([])
    expect(spawnedObjectDeployments(docWith(6, [tracked(100, 105)]))).toEqual([])
  })

  it("ne recompte pas une famille que les poses du document déploient déjà (artefact antérieur à la règle du 15/09)", () => {
    const doc = docWith(1, [tracked(100, 105)], {
      equipmentPlacements: [
        { family: 'sensor', id: '0x1', origin: 'deployed', owner: 540, t0: 100, t1: 120, x: 0, y: 0 },
      ],
    })
    expect(spawnedObjectDeployments(doc)).toEqual([])
  })
})

describe('branchements : zones, survol et révélation lisent les poses fabriquées', () => {
  const HOVER = { showUnnamed: false, showDropped: true, ...TIME }
  const campOfSlot = (slot: number) => (slot === 540 ? 0 : 1)

  it('la présence en zone suit la fenêtre du champ : dedans entre le repos et la fin lue, pas après', () => {
    const doc = docWith(5, [tracked(100, 105, { t1: 125, x: 3, y: 4 })])
    const scene = { placements: drawablePlacements(doc), campOfSlot }
    const at = (frame: number) => zonePresenceAt(scene, { slot: 7, x: 3.5, y: 4, frame }, TIME).repair
    expect(at(104)).toBe(false)
    expect(at(110)).toBe(true)
    expect(at(125)).toBe(true)
    expect(at(126)).toBe(false)
  })

  it('le survol désigne la pose fabriquée tant qu’elle est à l’écran, et plus ensuite', () => {
    const doc = docWith(1, [tracked(100, 105, { t1: 125, x: 5, y: 5 })])
    const placements = drawablePlacements(doc)
    const c = project({ x: 5, y: 5 }, VIEW)
    expect(placementAt(placements, VIEW, { ...HOVER, frame: 110 }, c)?.family).toBe('sensor')
    expect(placementAt(placements, VIEW, { ...HOVER, frame: 126 }, c)).toBeNull()
  })

  it('le capteur fabriqué révèle un adversaire dans sa portée, au nom de son porteur', () => {
    const doc = docWith(1, [tracked(100, 105, { t1: 205, x: 0, y: 0 })])
    const [p] = drawablePlacements(doc)
    const enemy: ReplayTrackReady = {
      slot: 3,
      team: -1,
      startFrame: 0,
      endFrame: 300,
      points: [
        { t: 0, x: 1, y: 1 },
        { t: 300, x: 1, y: 1 },
      ],
    }
    const reveals = sensorReveals([p], { lives: [enemy], campOfSlot }, { frame: 105, ...TIME })
    expect(reveals.map((r) => [r.slot, r.owner, r.ownerFrame])).toEqual([[3, 540, 105]])
  })
})

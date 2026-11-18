/**
 * Tests — reliefViewLogic : les décisions pures du fond Relief et de la coupe par étage des vues 2D sans panneau
 * « Vue » (Tactique, « Occupation du terrain »).
 */
import { describe, expect, it } from 'vitest'

import { CUT_BAND_U } from './relief'
import { fourBlocksScene, reliefData } from './relief/reliefScene.testutil'
import {
  isCutAtLevel,
  nearestChip,
  positionInCut,
  reliefViewRequest,
  splitPositionsByCut,
  viewBackgroundOptions,
  viewFloorState,
  zoneInCut,
  zonesInCut,
} from './reliefViewLogic'

describe('viewBackgroundOptions', () => {
  it('propose Plan, Rendu, Relief dans cet ordre', () => {
    expect(viewBackgroundOptions(true, true)).toEqual(['plan', 'rendu', 'relief'])
  })

  it('ne propose rien quand il n’y a pas de choix (le rendu seul)', () => {
    expect(viewBackgroundOptions(false, false)).toEqual([])
  })

  it('un relief seul suffit à faire un choix (Rendu, Relief)', () => {
    expect(viewBackgroundOptions(false, true)).toEqual(['rendu', 'relief'])
  })

  it('un plan seul suffit à faire un choix (Plan, Rendu)', () => {
    expect(viewBackgroundOptions(true, false)).toEqual(['plan', 'rendu'])
  })
})

describe('viewFloorState : l’état de la commande d’étage', () => {
  it('aucune commande tant que les données ne sont pas lues', () => {
    expect(viewFloorState(null)).toEqual({ state: 'none' })
  })

  it('des étages : les rangs, du plus bas au plus haut, et les bornes du curseur', () => {
    const state = viewFloorState(fourBlocksScene().data)
    expect(state.state).toBe('ready')
    if (state.state !== 'ready') return
    expect(state.chips).toEqual([
      { rank: 0, z: 0 },
      { rank: 1, z: 2.2 },
    ])
    expect(state.min).toBeLessThanOrEqual(0)
    expect(state.max).toBeGreaterThanOrEqual(2.2)
  })

  it('carte sans maillage de navigation : état « sans maillage », jamais de pastille', () => {
    const data = reliefData({ width: 4, height: 4, degradations: ['sans-maillage'] })
    expect(viewFloorState(data)).toEqual({ state: 'noMesh' })
  })

  it('un maillage sans étage détecté vaut « sans maillage »', () => {
    const data = reliefData({ width: 4, height: 4, levels: [] })
    expect(viewFloorState(data)).toEqual({ state: 'noMesh' })
  })
})

describe('la pastille retenue', () => {
  it('seule la hauteur exacte d’un étage retient sa pastille', () => {
    expect(isCutAtLevel(2.2, 2.2)).toBe(true)
    expect(isCutAtLevel(2.25, 2.2)).toBe(false)
    expect(isCutAtLevel(null, 0)).toBe(false)
  })

  it('nearestChip nomme l’étage le plus proche du curseur', () => {
    const chips = [
      { rank: 0, z: 0 },
      { rank: 1, z: 2.2 },
    ]
    expect(nearestChip(chips, 1.5)?.rank).toBe(1)
    expect(nearestChip(chips, 0.4)?.rank).toBe(0)
    expect(nearestChip([], 1)).toBeNull()
  })
})

describe('reliefViewRequest : la demande de rendu', () => {
  const base = { needed: true, turns: 1 as const, cut: null, lines: false }

  it('rien à calculer : pas de demande', () => {
    expect(reliefViewRequest({ ...base, needed: false, tintBase: null })).toBeNull()
  })

  it('le fond Relief veut l’image teintée, l’orientation règle la lumière, pas de masque', () => {
    expect(reliefViewRequest({ ...base, tintBase: '#6b7078', cut: 2.2 })).toEqual({
      turns: 1,
      cut: 2.2,
      base: '#6b7078',
      lines: false,
      masks: false,
    })
  })

  it('un fond Plan ou Rendu sans coupe ne veut aucun masque', () => {
    expect(reliefViewRequest({ ...base, tintBase: null, lines: true })).toEqual({
      turns: 0,
      cut: null,
      base: null,
      lines: true,
      masks: false,
    })
  })

  it('un fond Plan ou Rendu sous coupe veut les masques', () => {
    expect(reliefViewRequest({ ...base, tintBase: null, cut: 0 })?.masks).toBe(true)
  })
})

describe('zoneInCut : les zones nommées portent une hauteur', () => {
  const zone = { zBottom: 1, zTop: 3 }

  it('sans coupe, toute zone reste', () => {
    expect(zoneInCut(zone, null)).toBe(true)
  })

  it('la zone reste quand sa tranche croise la bande de la coupe', () => {
    expect(zoneInCut(zone, 2)).toBe(true)
    expect(zoneInCut(zone, 3 + CUT_BAND_U)).toBe(true)
    expect(zoneInCut(zone, 1 - CUT_BAND_U)).toBe(true)
  })

  it('elle disparaît quand sa tranche est hors de la bande', () => {
    expect(zoneInCut(zone, 3 + CUT_BAND_U + 0.01)).toBe(false)
    expect(zoneInCut(zone, 1 - CUT_BAND_U - 0.01)).toBe(false)
  })

  it('une tranche aux bornes inversées se lit dans le bon sens', () => {
    expect(zoneInCut({ zBottom: 3, zTop: 1 }, 2)).toBe(true)
    expect(zoneInCut({ zBottom: 3, zTop: 1 }, 6)).toBe(false)
  })

  it('une tranche non finie est gardée : mieux vaut une zone de trop qu’une zone perdue', () => {
    expect(zoneInCut({ zBottom: Number.NaN, zTop: 3 }, 9)).toBe(true)
  })

  it('zonesInCut rend la MÊME liste sans coupe, et filtre avec', () => {
    const zones = [
      { name: 'bas', zBottom: 0, zTop: 1 },
      { name: 'haut', zBottom: 4, zTop: 6 },
    ]
    expect(zonesInCut(zones, null)).toBe(zones)
    expect(zonesInCut(zones, 5).map((z) => z.name)).toEqual(['haut'])
    expect(zonesInCut(zones, 0.5).map((z) => z.name)).toEqual(['bas'])
  })
})

describe('splitPositionsByCut : les positions d’« Occupation du terrain »', () => {
  const mesh = fourBlocksScene().data.floors.mesh
  const positions = [
    { x: 0.5, y: 0.5, z: 0 }, // sol à 0
    { x: 1.5, y: 0.5, z: 2.2 }, // sol à 2,2
    { x: 2.5, y: 0.5, z: 0 }, // sol à 0
    { x: 3.5, y: 0.5, z: 2.2 }, // sol à 2,2
  ]

  it('sans coupe ou sans maillage, tout reste (même liste) et rien n’est masqué', () => {
    expect(splitPositionsByCut(positions, mesh, null)).toEqual({ kept: positions, hidden: 0 })
    expect(splitPositionsByCut(positions, null, 0).kept).toBe(positions)
  })

  it('l’étage bas masque les positions de l’étage haut, et en donne le compte', () => {
    const { kept, hidden } = splitPositionsByCut(positions, mesh, 0)
    expect(kept).toEqual([positions[0], positions[2]])
    expect(hidden).toBe(2)
  })

  it('l’étage haut masque celles du bas', () => {
    const { kept, hidden } = splitPositionsByCut(positions, mesh, 2.2)
    expect(kept).toEqual([positions[1], positions[3]])
    expect(hidden).toBe(2)
  })

  it('un joueur en l’air garde l’étage de ses pieds : un saut ne le fait pas disparaître', () => {
    const jump = { x: 0.5, y: 0.5, z: 0.9 }
    expect(positionInCut(jump, mesh, 0)).toBe(true)
    expect(positionInCut(jump, mesh, 2.2)).toBe(false)
  })

  it('sans sol sous la position, on juge sur sa hauteur', () => {
    const outside = { x: 50, y: 50, z: 2.2 }
    expect(positionInCut(outside, mesh, 2.2)).toBe(true)
    expect(positionInCut(outside, mesh, 0)).toBe(false)
  })
})

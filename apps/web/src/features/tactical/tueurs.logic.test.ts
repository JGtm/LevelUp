import { describe, expect, it } from 'vitest'

import type { TacticalContribution } from '@/lib/api/types'

import {
  basculerChoix,
  dimensionsDeLaCouche,
  distanceEnMetres,
  formeDuRole,
  misEnAvant,
  pointLePlusProche,
  pointsDeLaCouche,
  rolesPresents,
  roleDeLaFace,
} from './tueurs.logic'

function contribution(extra: Partial<TacticalContribution> = {}): TacticalContribution {
  return {
    match_id: 'm1',
    instant_ms: 1000,
    clock: 'match',
    xuid: '1',
    match_started_at: '2026-09-01T22:30:00Z',
    replay_available: true,
    ...extra,
  }
}

describe('roleDeLaFace — la face décide, jamais la lecture', () => {
  it('mort -> tueur, frag -> victime, le reste -> aucun', () => {
    expect(roleDeLaFace('mort')).toBe('tueur')
    expect(roleDeLaFace('frag')).toBe('victime')
    expect(roleDeLaFace('entree')).toBeNull()
    expect(roleDeLaFace('reapparition')).toBeNull()
    expect(roleDeLaFace(undefined)).toBeNull()
  })
})

describe('distanceEnMetres', () => {
  it('distance 2D arrondie au mètre', () => {
    expect(distanceEnMetres({ point: { x: 0, y: 0 }, autre_point: { x: 3, y: 4 } })).toBe(5)
    expect(distanceEnMetres({ point: { x: 0, y: 0 }, autre_point: { x: 23.4, y: 0 } })).toBe(23)
    expect(distanceEnMetres({ point: { x: 0, y: 0 }, autre_point: { x: 23.6, y: 0 } })).toBe(24)
  })
  it('absente quand un point manque', () => {
    expect(distanceEnMetres({ point: { x: 0, y: 0 } })).toBeNull()
    expect(distanceEnMetres({ autre_point: { x: 0, y: 0 } })).toBeNull()
    expect(distanceEnMetres({})).toBeNull()
  })
})

describe('pointsDeLaCouche', () => {
  const p = { point: { x: 1, y: 1 }, autre_point: { x: 4, y: 5 } }
  it('mort -> tueur, frag -> victime, avec l’index de la contribution', () => {
    const pts = pointsDeLaCouche([contribution({ face: 'mort', ...p }), contribution({ face: 'frag', ...p })])
    expect(pts.map((x) => [x.index, x.role, x.distanceM])).toEqual([
      [0, 'tueur', 5],
      [1, 'victime', 5],
    ])
    expect(pts[0]?.vers).toEqual({ x: 4, y: 5 })
    expect(pts[0]?.de).toEqual({ x: 1, y: 1 })
  })
  it('ni autre joueur (temps, routes) ni point manquant : rien', () => {
    const pts = pointsDeLaCouche([
      contribution({ face: 'entree', ...p }),
      contribution({ face: 'reapparition' }),
      contribution({ face: 'mort', point: p.point }),
      contribution({ face: 'frag', autre_point: p.autre_point }),
    ])
    expect(pts).toEqual([])
  })
  it('les rôles présents suivent l’ordre de la légende', () => {
    const pts = pointsDeLaCouche([contribution({ face: 'frag', ...p }), contribution({ face: 'mort', ...p })])
    expect(rolesPresents(pts)).toEqual(['tueur', 'victime'])
    expect(rolesPresents(pts.slice(0, 1))).toEqual(['victime'])
    expect(rolesPresents([])).toEqual([])
  })
})

describe('pointLePlusProche — test de survol en px écran', () => {
  const projeter = (m: { x: number; y: number }) => ({ x: m.x * 10, y: m.y * 10 })
  const pts = pointsDeLaCouche([
    contribution({ face: 'mort', point: { x: 0, y: 0 }, autre_point: { x: 10, y: 10 } }),
    contribution({ face: 'frag', point: { x: 0, y: 0 }, autre_point: { x: 11, y: 10 } }),
  ])
  it('le plus proche dans le rayon', () => {
    expect(pointLePlusProche(pts, { x: 100, y: 100 }, projeter, 9)).toBe(0)
    expect(pointLePlusProche(pts, { x: 109, y: 100 }, projeter, 9)).toBe(1)
  })
  it('aucun hors du rayon', () => {
    expect(pointLePlusProche(pts, { x: 140, y: 100 }, projeter, 9)).toBeNull()
    expect(pointLePlusProche([], { x: 0, y: 0 }, projeter, 9)).toBeNull()
  })
})

describe('dimensionsDeLaCouche — la taille suit l’échelle du plan (modèle de taille du rejeu 2D)', () => {
  it('croît avec l’échelle : un plan zoomé ou une petite carte grossit les points', () => {
    const grande = dimensionsDeLaCouche(4) // grande carte, plan entier
    const zoome = dimensionsDeLaCouche(16) // même carte zoomée x4
    expect(zoome.rayonPoint).toBeGreaterThan(grande.rayonPoint)
    expect(zoome.epaisseurTrait).toBeGreaterThanOrEqual(grande.epaisseurTrait)
    expect(zoome.rayonSurvol).toBeGreaterThanOrEqual(grande.rayonSurvol)
  })
  it('plancher et plafond en px', () => {
    // Sous le plancher d'échelle du rejeu, la taille ne descend plus : un palier, pas un zéro.
    const mini = dimensionsDeLaCouche(0.001)
    expect(mini).toEqual(dimensionsDeLaCouche(1))
    expect(mini.rayonPoint).toBeGreaterThanOrEqual(3)
    expect(mini.epaisseurTrait).toBeGreaterThanOrEqual(1)
    expect(mini.rayonSurvol).toBeGreaterThanOrEqual(9)
    const maxi = dimensionsDeLaCouche(10_000)
    expect(maxi.rayonPoint).toBe(13.5)
    expect(maxi.epaisseurTrait).toBe(3)
    expect(maxi.rayonSurvol).toBe(24)
  })
  it('la mise en avant grossit le point et le trait', () => {
    const d = dimensionsDeLaCouche(8)
    expect(d.rayonPointAvant).toBeGreaterThan(d.rayonPoint)
    expect(d.epaisseurTraitAvant).toBeGreaterThan(d.epaisseurTrait)
  })
})

describe('choix d’une contribution', () => {
  it('le survol prime sur le choix, à défaut le choix tient', () => {
    expect(misEnAvant(2, 5)).toBe(2)
    expect(misEnAvant(null, 5)).toBe(5)
    expect(misEnAvant(null, null)).toBeNull()
  })
  it('re-clic = désélection', () => {
    expect(basculerChoix(null, 3)).toBe(3)
    expect(basculerChoix(3, 3)).toBeNull()
    expect(basculerChoix(3, 4)).toBe(4)
  })
})

describe('formeDuRole — la forme porte la distinction, pas la couleur', () => {
  it('tueur = losange, victime = rond', () => {
    expect(formeDuRole('tueur')).toBe('losange')
    expect(formeDuRole('victime')).toBe('rond')
  })
})

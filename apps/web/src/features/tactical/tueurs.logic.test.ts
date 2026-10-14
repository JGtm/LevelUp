import { describe, expect, it } from 'vitest'

import type { TacticalContribution } from '@/lib/api/types'

import { distanceEnMetres, pointLePlusProche, pointsDeLaCouche, rolesPresents, roleDeLaFace } from './tueurs.logic'

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
    expect(pointLePlusProche([], { x: 0, y: 0 }, projeter)).toBeNull()
  })
})

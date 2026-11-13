/**
 * moyenneGlissante (Escouade, portée par rôle) avant et après son passage sur `movingMean` (étape 7.8) :
 * l'ancien calcul est gardé ici comme référence, les deux doivent rendre les mêmes nombres, au bit près.
 */
import { describe, expect, it } from 'vitest'

import { moyenneGlissante, type PointPortee } from './squadRangeRoles.logic'

function ancienne(points: PointPortee[], fenetre: number): (number | null)[] {
  return points.map((_, i) => {
    if (i < fenetre - 1) return null
    const pleins = points.slice(i - fenetre + 1, i + 1).filter((p) => p.plein)
    if (pleins.length === 0) return null
    return pleins.reduce((s, p) => s + p.ecartM, 0) / pleins.length
  })
}

function pt(plein: boolean, ecartM: number): PointPortee {
  return { ordre: 0, matchId: 'm', xuid: 'x', gamertag: 'g', medianeM: 0, ecartM, mesures: plein ? 5 : 1, plein } as PointPortee
}

const CAS: Array<[string, PointPortee[], number]> = [
  ['série vide', [], 5],
  ['fenêtre jamais pleine', [pt(true, 1), pt(true, 2)], 5],
  ['tous pleins', [pt(true, 0.1), pt(true, 0.2), pt(true, 0.3), pt(true, 0.4), pt(true, 0.5), pt(true, 0.6)], 3],
  ['creux mêlés', [pt(true, 1.25), pt(false, 99), pt(true, -0.5), pt(false, 7), pt(false, 8), pt(true, 2.75), pt(true, 3.5)], 3],
  ['fenêtre sans point plein', [pt(false, 1), pt(false, 2), pt(false, 3), pt(true, 4)], 3],
  ['fenêtre de 1', [pt(true, 0.7), pt(false, 3), pt(true, 1.1)], 1],
  ['décimales non représentables', [pt(true, 0.1), pt(true, 0.7), pt(true, 0.2), pt(true, 1e-9), pt(true, 1e9)], 2],
]

describe('moyenneGlissante (portée) : mêmes nombres qu’avant 7.8', () => {
  it.each(CAS)('%s', (_label, points, fenetre) => {
    const attendu = ancienne(points, fenetre)
    const obtenu = moyenneGlissante(points, fenetre)
    expect(obtenu).toHaveLength(attendu.length)
    obtenu.forEach((v, i) => expect(Object.is(v, attendu[i])).toBe(true))
  })
})

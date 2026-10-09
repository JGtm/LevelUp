/**
 * vehicles.logic.test.ts — la ressource « véhicules » dans les modèles purs de l'onglet Emprise
 * (lot L7.4 du plan PLAN_EMPRISE_VEHICULES_2026-09-28) : une entrée de liste de plus (ordre, bilan,
 * fil, fiches, grille), indépendante du film, case vide distincte d'un zéro (D8), la barre épaisse
 * sur tous les frags et le rendement lu du Go (D9), nom des familles.
 */
import { describe, expect, it } from 'vitest'

import { RESOURCE_ORDER, RESOURCE_VEHICLE, buildControlRows, buildMatchGrid, buildPickupSheets, buildResourceFil, empriseMatchIndex } from './emprise.logic'
import { EMPRISE_2209, HISTORY_2209 } from './emprise.fixtures'
import { buildHabitView } from './habit.logic'
import { buildProductionRows, buildYieldRows } from './production.logic'
import { VEHICLES_2209 } from './vehicles.fixtures'
import { vehicleFamilyName } from './vehicles.logic'

const name = (o: { key: string; label?: string }) => vehicleFamilyName(o.key, o.label, 'Véhicule inconnu')

describe('RESOURCE_ORDER — une entrée de plus', () => {
  it('bonus, armes spéciales, véhicules, râteliers', () => {
    expect(RESOURCE_ORDER).toEqual(['powerup', 'power_weapon', 'vehicle', 'rack'])
  })

  it('sans véhicules dans le bloc, rien n’est rendu pour eux', () => {
    expect(buildControlRows(EMPRISE_2209).map((r) => r.resource)).not.toContain(RESOURCE_VEHICLE)
    expect(buildResourceFil(EMPRISE_2209, empriseMatchIndex(HISTORY_2209)).resources).not.toContain(RESOURCE_VEHICLE)
    expect(buildProductionRows(EMPRISE_2209).map((r) => r.resource)).not.toContain(RESOURCE_VEHICLE)
  })
})

describe('bilan, fil et fiches', () => {
  it('le bilan compte 5 / 3 prises de véhicule, entre les armes spéciales et le reste', () => {
    const rows = buildControlRows(VEHICLES_2209)
    expect(rows.map((r) => [r.resource, r.us, r.them])).toEqual([
      ['powerup', 12, 8],
      ['power_weapon', 23, 29],
      ['vehicle', 5, 3],
    ])
    expect(rows[2].share).toBeCloseTo(5 / 8)
  })

  it('le fil lit les véhicules indépendamment du film : Detachment (sans film) a son point, Shogun (non mesuré) non', () => {
    const fil = buildResourceFil(VEHICLES_2209, empriseMatchIndex(HISTORY_2209))
    expect(fil.resources).toEqual(['powerup', 'power_weapon', 'vehicle'])
    const byMap = Object.fromEntries(fil.matches.map((m) => [m.map, m.points.vehicle]))
    expect(byMap.Starboard).toMatchObject({ us: 3, them: 1 })
    expect(byMap.Detachment).toMatchObject({ us: 2, them: 2 })
    expect(byMap.Shogun).toBeNull()
    expect(byMap.Curfew).toBeNull()
    // Cumul : 3 sur 4, puis 5 sur 8.
    expect(byMap.Detachment?.cumulative).toBeCloseTo(5 / 8)
  })

  it('fiches : le Warthog, la tourelle fixe (libellé du titre) et le châssis inconnu ; pastilles toutes pleines (D3)', () => {
    const sheets = buildPickupSheets(VEHICLES_2209, name)
    const section = sheets.sections.find((s) => s.resource === RESOURCE_VEHICLE)!
    expect(section.lines.map((l) => name(l.object))).toEqual(['Warthog', 'Tourelle fixe', 'Véhicule inconnu'])
    const warthog = section.lines[0]
    expect(warthog.taken).toEqual([2, 1, 0, 0])
    expect(warthog.kept.every((n) => n === 0) && warthog.dropped.every((n) => n === 0)).toBe(true)
    expect(section.totals).toEqual([3, 1, 0, 1])
    // Les pertes du bilan ne parlent que des bonus.
    expect(sheets.losses?.us.taken).toBe(12)
  })
})

describe('grille match par match', () => {
  const grid = buildMatchGrid(VEHICLES_2209, empriseMatchIndex(HISTORY_2209))
  const section = grid.sections.find((s) => s.resource === RESOURCE_VEHICLE)!
  const col = (map: string) => grid.columns.findIndex((c) => c.map === map)

  it('une ligne de synthèse puis une par famille, du plus pris au moins pris', () => {
    expect(section.items.map((r) => r.object!.key)).toEqual(['warthog', 'tourelle_fixe', 'unknown', 'banshee'])
  })

  it('case vide (Shogun, non lu) ≠ zéro (Curfew, « rien ») ; sans film mais lu (Detachment) = une valeur', () => {
    expect(section.summary!.cells[col('Shogun')]).toEqual({ kind: 'blank' })
    expect(section.summary!.cells[col('Curfew')]).toEqual({ kind: 'none' })
    expect(section.summary!.cells[col('Starboard')]).toMatchObject({ kind: 'value', us: 3, them: 1 })
    expect(section.summary!.cells[col('Detachment')]).toMatchObject({ kind: 'value', us: 2, them: 2 })
    // Les lignes qui viennent du film, elles, restent « sans film » à Detachment.
    const bonus = grid.sections.find((s) => s.resource === 'powerup')!
    expect(bonus.summary!.cells[col('Detachment')].kind).toBe('blank')
  })

  it('une case d’objet dit qui chez nous a pris', () => {
    const warthog = section.items[0].cells[col('Starboard')]
    expect(warthog).toMatchObject({ kind: 'value', us: 3, them: 1 })
    expect(warthog.kind === 'value' && warthog.who).toEqual([
      { xuid: 'x-jgtm', taken: 2 },
      { xuid: 'x-choco', taken: 1 },
    ])
  })
})

describe('frags et rendement (D5, D9)', () => {
  it('la barre épaisse garde tous les frags (14 / 9), la fine le temps à bord', () => {
    const row = buildProductionRows(VEHICLES_2209).find((r) => r.resource === RESOURCE_VEHICLE)!
    expect(row.kills).toEqual({ us: 14, them: 9 })
    expect(row.exposure).toEqual({ kind: 'aboard_ms', value: { us: 210_000, them: 100_000 } })
  })

  it('le rendement est celui du Go, calculé sur les frags appariés (8 et 3), pas sur 14 et 9', () => {
    const row = buildYieldRows(VEHICLES_2209).find((r) => r.resource === RESOURCE_VEHICLE)!
    expect(row.us).toBeCloseTo(8 / 3.5)
    expect(row.them).toBeCloseTo(3 / (100 / 60))
    expect(row.us).not.toBeCloseTo(14 / 3.5)
    expect(row.gap).toBeCloseTo(0.2698, 3)
  })

  it('habitude : notre part des prises de véhicule sur chaque soirée, médiane dès trois soirées', () => {
    const view = buildHabitView(VEHICLES_2209)
    if (view.kind !== 'chart') throw new Error('habitude attendue')
    expect(view.resources).toEqual(['powerup', 'power_weapon', 'vehicle'])
    expect(view.points[view.points.length - 1].shares.vehicle).toBeCloseTo(62.5)
    expect(view.medians.vehicle).not.toBeNull()
  })
})

describe('nom des familles', () => {
  it('libellé du titre, sinon le nom propre tiré de la clé, sinon « Véhicule inconnu »', () => {
    expect(vehicleFamilyName('tourelle_fixe', 'Tourelle fixe', 'Inconnu')).toBe('Tourelle fixe')
    expect(vehicleFamilyName('warthog', undefined, 'Inconnu')).toBe('Warthog')
    expect(vehicleFamilyName('rocket_hog', '', 'Inconnu')).toBe('Rocket Hog')
    expect(vehicleFamilyName('unknown', undefined, 'Inconnu')).toBe('Inconnu')
    expect(vehicleFamilyName('', undefined, 'Inconnu')).toBe('Inconnu')
  })

})

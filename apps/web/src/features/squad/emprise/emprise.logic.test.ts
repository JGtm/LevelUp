/**
 * emprise.logic.test.ts — les modèles purs de l'onglet Emprise sur la soirée témoin du 22/09
 * (chiffres de la maquette de l'onglet) et leurs cas limites (ressource absente, match sans
 * historique, niveaux de socle non mesurés, liste de ressources pilotée par le bloc).
 */
import { describe, expect, it } from 'vitest'

import type { SquadEmpriseBlock } from '@/lib/api/types'

import {
  buildControlRows,
  buildMatchGrid,
  buildPickupSheets,
  buildResourceFil,
  type GridCell,
} from './emprise.logic'
import { EMPRISE_2209, HISTORY_2209, WEAPONS, XUID } from './emprise.fixtures'

const nameOf = (o: { key: string; label?: string }) =>
  o.key === 'powerup_camo' ? 'Camouflage' : o.key === 'powerup_overshield' ? 'Surbouclier' : (o.label ?? o.key)

describe('buildControlRows — Contrôle des ressources', () => {
  it('22/09 : bonus 12 / 8 (60 %), armes spéciales 23 / 29, dans l’ordre des ressources', () => {
    const rows = buildControlRows(EMPRISE_2209)
    expect(rows.map((r) => [r.resource, r.us, r.them])).toEqual([
      ['powerup', 12, 8],
      ['power_weapon', 23, 29],
    ])
    expect(rows[0].share).toBeCloseTo(0.6)
    expect(rows[1].share).toBeCloseTo(23 / 52)
  })

  it('la liste suit le bloc : une ressource absente n’a pas de piste, une inconnue n’est pas rendue', () => {
    const block: SquadEmpriseBlock = {
      ...EMPRISE_2209,
      resources: [
        { resource: 'vehicle_future', taken: { us: 3, them: 1 }, matches_measured: 6 },
        { resource: 'power_weapon', taken: { us: 23, them: 29 }, matches_measured: 6 },
        { resource: 'powerup', taken: { us: 0, them: 0 }, matches_measured: 6 },
      ],
    }
    expect(buildControlRows(block).map((r) => r.resource)).toEqual(['power_weapon'])
  })
})

describe('buildResourceFil — au fil de la session', () => {
  const fil = buildResourceFil(EMPRISE_2209, HISTORY_2209)

  it('sept matchs dans l’ordre, joints à l’historique (Starboard : victoire 3–0, Domination)', () => {
    expect(fil.resources).toEqual(['powerup', 'power_weapon'])
    expect(fil.matches.map((m) => m.map)).toEqual([
      'Starboard', 'Curfew', 'Origin', 'Solution', 'Detachment', 'Shogun', 'Catalyst',
    ])
    expect(fil.matches[0]).toMatchObject({ outcome: 'win', score: '3–0', dominance: 1, mode: 'Drapeau' })
  })

  it('les parts par match de la maquette (FIL_SERIES), null sans la ressource ou sans film', () => {
    const pairs = (r: string) => fil.matches.map((m) => (m.points[r] ? [m.points[r]!.us, m.points[r]!.them] : null))
    expect(pairs('powerup')).toEqual([[5, 2], [4, 0], null, null, null, [2, 3], [1, 3]])
    expect(pairs('power_weapon')).toEqual([[2, 4], [6, 2], [5, 7], [2, 0], null, [3, 5], [5, 11]])
  })

  it('le cumul = nos prises / toutes les prises depuis le premier match ; fin = le bilan (60 %, 23 sur 52)', () => {
    const last = fil.matches[6]
    expect(last.points.powerup).toMatchObject({ cumUs: 12, cumTotal: 20 })
    expect(last.points.powerup!.cumulative).toBeCloseTo(0.6)
    expect(last.points.power_weapon).toMatchObject({ cumUs: 23, cumTotal: 52 })
    expect(fil.matches[1].points.powerup!.cumulative).toBeCloseTo(9 / 11)
  })

  it('un match sans ligne d’historique garde sa place, sans résultat ni erreur', () => {
    const f = buildResourceFil(EMPRISE_2209, HISTORY_2209.filter((h) => h.match_id !== 'm3'))
    expect(f.matches).toHaveLength(7)
    expect(f.matches[2]).toMatchObject({ map: '', outcome: null, score: null, dominance: undefined })
    expect(f.matches[2].points.power_weapon).toMatchObject({ us: 5, them: 7 })
  })
})

describe('buildPickupSheets — Répartition des prises dans l’escouade', () => {
  const sheets = buildPickupSheets(EMPRISE_2209, nameOf)

  it('une fiche par joueur puis le reste du camp ; bonus puis armes spéciales, pas les râteliers', () => {
    expect(sheets.owners.map((o) => o.xuid)).toEqual([XUID.jgtm, XUID.choco, XUID.madina, null])
    expect(sheets.sections.map((s) => s.resource)).toEqual(['powerup', 'power_weapon'])
  })

  it('fiches du 22/09 (ROLES) : bonus 3 / 3 / 5 / 1, armes spéciales 9 / 3 / 3 / 8', () => {
    expect(sheets.sections[0].totals).toEqual([3, 3, 5, 1])
    expect(sheets.sections[1].totals).toEqual([9, 3, 3, 8])
    expect(sheets.sections[0].camp).toBe(12)
    expect(sheets.sections[1].camp).toBe(23)
  })

  it('mêmes lignes dans le même ordre : prises de notre camp décroissantes, puis nom', () => {
    expect(sheets.sections[0].lines.map((l) => nameOf(l.object))).toEqual(['Camouflage', 'Surbouclier'])
    const power = sheets.sections[1].lines.map((l) => nameOf(l.object))
    expect(power[0]).toBe(WEAPONS.a1000001)
    expect(power).toHaveLength(7)
    const spnkr = sheets.sections[1].lines[0]
    expect(spnkr.taken).toEqual([5, 2, 0, 2])
  })

  it('bonus perdus par joueur : Chocoboflor garde un camouflage, Madina97294 en lâche un', () => {
    const camo = sheets.sections[0].lines.find((l) => l.object.key === 'powerup_camo')!
    expect(camo.taken).toEqual([1, 1, 4, 0])
    expect(camo.kept).toEqual([0, 1, 0, 0])
    expect(camo.dropped).toEqual([0, 0, 1, 0])
  })

  it('ressource dominante : la part du camp la plus forte (JGtm armes spéciales, Madina97294 bonus)', () => {
    expect(sheets.dominant).toEqual(['power_weapon', 'powerup', 'powerup', 'power_weapon'])
  })

  it('bonus perdus : 2 sur 12 pour nous, 2 sur 8 pour eux', () => {
    expect(sheets.losses).toEqual({ us: { lost: 2, taken: 12 }, them: { lost: 2, taken: 8 } })
  })
})

describe('buildMatchGrid — match par match', () => {
  const grid = buildMatchGrid(EMPRISE_2209, HISTORY_2209)
  const cellText = (c: GridCell) => (c.kind === 'value' ? `${c.us}–${c.them}` : c.kind)

  it('bonus, armes spéciales (avec les frags obtenus avec), puis armes de râtelier', () => {
    expect(grid.sections.map((s) => s.resource)).toEqual(['powerup', 'power_weapon', 'rack'])
    expect(grid.sections[1].kills).not.toBeNull()
    expect(grid.sections[0].kills).toBeNull()
    expect(grid.sections[2].items).toHaveLength(13)
  })

  it('synthèse bonus : « 5–2 » à Starboard, « — » sans bonus sur la carte, « sans film » à Detachment', () => {
    expect(grid.sections[0].summary!.cells.map(cellText)).toEqual(['5–2', '4–0', 'none', 'none', 'nofilm', '2–3', '1–3'])
  })

  it('frags aux armes spéciales : la feuille de match, lue même sans film', () => {
    expect(grid.sections[1].kills!.cells.map(cellText)).toEqual(['0–4', '12–4', '8–11', '5–0', '9–13', '4–10', '9–12'])
  })

  it('« qui chez nous » par case d’objet ; socles vidés sur les bonus', () => {
    const surb = grid.sections[0].items.find((r) => r.object?.key === 'powerup_overshield')!
    const c = surb.cells[0]
    expect(c.kind).toBe('value')
    if (c.kind !== 'value') return
    expect(c.padsEmptied).toBe(10)
    expect(c.who).toEqual([
      { xuid: XUID.jgtm, taken: 2 },
      { xuid: XUID.choco, taken: 2 },
      { xuid: XUID.madina, taken: 1 },
    ])
  })

  it('niveaux de socle non mesurés : armes spéciales et râteliers « non classé », bonus lisibles', () => {
    const block: SquadEmpriseBlock = {
      ...EMPRISE_2209,
      matches: EMPRISE_2209.matches!.map((m) => (m.match_id === 'm2' ? { ...m, tiers: 'not_measured' } : m)),
    }
    const g = buildMatchGrid(block, HISTORY_2209)
    expect(g.sections[0].summary!.cells[1]).toMatchObject({ kind: 'value', us: 4, them: 0 })
    expect(g.sections[1].summary!.cells[1]).toEqual({ kind: 'untiered', tiers: 'not_measured' })
    expect(g.sections[2].items[0].cells[1]).toEqual({ kind: 'untiered', tiers: 'not_measured' })
  })
})

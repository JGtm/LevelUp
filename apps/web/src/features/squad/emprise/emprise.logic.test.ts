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
  empriseMatchIndex,
  squadPickupSheets,
  type GridCell,
} from './emprise.logic'
import { EMPRISE_2209, HISTORY_2209, WEAPONS, XUID } from './emprise.fixtures'

/**
 * Le 22/09 où un match FILMÉ a perdu notre camp (`team_known = false`). Ses prises restent dans le
 * bloc : le web ne doit pas compter sur leur absence pour les écarter.
 */
function campInconnu(matchId: string): SquadEmpriseBlock {
  return { ...EMPRISE_2209, matches: EMPRISE_2209.matches!.map((m) => (m.match_id === matchId ? { ...m, team_known: false } : m)) }
}

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
  const fil = buildResourceFil(EMPRISE_2209, empriseMatchIndex(HISTORY_2209))

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
    const f = buildResourceFil(EMPRISE_2209, empriseMatchIndex(HISTORY_2209.filter((h) => h.match_id !== 'm3')))
    expect(f.matches).toHaveLength(7)
    expect(f.matches[2]).toMatchObject({ map: '', outcome: null, score: null, dominance: undefined })
    expect(f.matches[2].points.power_weapon).toMatchObject({ us: 5, them: 7 })
  })

  it('constat R2 (revue L6.1) : un match filmé au camp inconnu n’a aucun point, comme un match sans donnée ; le cumul l’ignore', () => {
    const f = buildResourceFil(campInconnu('m2'), empriseMatchIndex(HISTORY_2209))
    expect(f.matches).toHaveLength(7)
    expect(f.matches[1].points).toEqual({ powerup: null, power_weapon: null })
    expect(f.matches[6].points.powerup).toMatchObject({ cumUs: 8, cumTotal: 16 })
    expect(f.matches[6].points.power_weapon).toMatchObject({ cumUs: 17, cumTotal: 44 })
  })
})

describe('squadPickupSheets — les fiches de l’escouade seule (Escouade › Emprise)', () => {
  const all = buildPickupSheets(EMPRISE_2209, nameOf)
  const only = squadPickupSheets(all)

  it('pas de fiche du reste du camp ; totaux, dominantes et pertes alignés sur les fiches restantes', () => {
    expect(only.owners.map((o) => o.xuid)).toEqual([XUID.jgtm, XUID.choco, XUID.madina])
    expect(only.sections[0].totals).toEqual([3, 3, 5])
    expect(only.dominant).toEqual(['power_weapon', 'powerup', 'powerup'])
    expect(only.losses).toEqual(all.losses)
  })

  it('un objet que seul le reste du camp a pris se retire ; le compte du camp reste celui de l’équipe', () => {
    const restOnly = all.sections[1].lines.filter((l) => l.taken.slice(0, 3).every((n) => n === 0))
    expect(restOnly.length).toBeGreaterThan(0)
    const keys = only.sections[1].lines.map((l) => l.object.key)
    for (const l of restOnly) expect(keys).not.toContain(l.object.key)
    expect(only.sections[1].lines).toHaveLength(all.sections[1].lines.length - restOnly.length)
    expect(only.sections[1].camp).toBe(23)
    expect(only.sections[1].lines[0].taken).toEqual([5, 2, 0])
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

  it('`resources` (Vue match) : les sections dans cet ordre, râteliers compris', () => {
    const rack = { resource: 'rack', key: 'vk78', label: 'VK78 Commando', taken: { us: 2, them: 0 }, squad: [{ xuid: XUID.jgtm, taken: 2 }] }
    const block: SquadEmpriseBlock = { ...EMPRISE_2209, objects: [...(EMPRISE_2209.objects ?? []), rack] }
    const s = buildPickupSheets(block, nameOf, ['rack', 'powerup'])
    expect(s.sections.map((x) => x.resource)).toEqual(['rack', 'powerup'])
    expect(s.sections[0].lines.find((l) => l.object.key === 'vk78')?.taken).toEqual([2, 0, 0, 0])
    // Sans `resources`, le bilan de l'onglet : pas de râteliers.
    expect(buildPickupSheets(block, nameOf).sections.map((x) => x.resource)).toEqual(['powerup', 'power_weapon'])
  })
})

describe('buildMatchGrid — match par match', () => {
  const grid = buildMatchGrid(EMPRISE_2209, empriseMatchIndex(HISTORY_2209))
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
    const g = buildMatchGrid(block, empriseMatchIndex(HISTORY_2209))
    expect(g.sections[0].summary!.cells[1]).toMatchObject({ kind: 'value', us: 4, them: 0 })
    expect(g.sections[1].summary!.cells[1]).toEqual({ kind: 'untiered', tiers: 'not_measured' })
    expect(g.sections[2].items[0].cells[1]).toEqual({ kind: 'untiered', tiers: 'not_measured' })
  })
})

describe('buildMatchGrid — camp inconnu (constat R2 de la revue L6.1)', () => {
  const g = buildMatchGrid(campInconnu('m2'), empriseMatchIndex(HISTORY_2209))

  it('filmé au camp inconnu : « camp inconnu » sur toutes les lignes lues au film, jamais « rien à prendre »', () => {
    expect(g.sections[0].summary!.cells[1]).toEqual({ kind: 'noteam' })
    expect(g.sections[1].summary!.cells[1]).toEqual({ kind: 'noteam' })
    for (const s of g.sections) for (const row of s.items) expect(row.cells[1]).toEqual({ kind: 'noteam' })
  })

  it('« sans film » garde la priorité : Detachment reste « sans film »', () => {
    const g2 = buildMatchGrid(campInconnu('m5'), empriseMatchIndex(HISTORY_2209))
    expect(g2.sections[0].summary!.cells[4]).toEqual({ kind: 'nofilm' })
  })
})

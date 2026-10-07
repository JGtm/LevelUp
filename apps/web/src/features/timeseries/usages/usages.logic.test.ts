/**
 * usages.logic.test.ts — les modèles purs de l'onglet « Usages » des Séries temporelles sur le
 * périmètre témoin (chiffres d'illustration de la maquette v4) et leurs cas limites : index des
 * matchs depuis `match_rows`, grille par carte (colonne de repli comprise), Mes prises, Équipement,
 * Mes vies, couverture, et le prédicat unique des blocs (page et état vide).
 */
import { describe, expect, it } from 'vitest'

import { block2209 } from '@/features/squad/objectif/objectif.fixtures'
import type { TimeseriesPageResponse } from '@/lib/api/types'

import { MATCH_ROWS, ME, lives, soloEmprise, soloEmpriseSansFilm } from './usages.fixtures'
import {
  buildEquipmentRows,
  buildLivesModel,
  buildMapGrid,
  buildMinePickups,
  timeseriesMatchIndex,
  usagesSections,
} from './usages.logic'

const nameOf = (o: { key: string; label?: string }) => o.label ?? o.key

describe('timeseriesMatchIndex — les matchs de la page, par match_id', () => {
  const index = timeseriesMatchIndex(MATCH_ROWS)

  it('date, carte (nom FR d’abord), résultat ; ni score ni dominance (pas d’encoche sur une période)', () => {
    expect(index.get('m1')).toEqual({
      matchId: 'm1',
      startTime: '2026-07-03T19:00:00Z',
      map: 'Carte Alpha',
      mode: 'Liste témoin',
      outcome: 'win',
      score: null,
      dominance: undefined,
    })
    expect(index.get('m2')).toMatchObject({ map: 'Carte Alpha', outcome: 'loss' })
    expect(index.get('m4')).toMatchObject({ map: 'Carte Charlie (fr)', outcome: 'dnf' })
  })
})

describe('buildMapGrid — une colonne par carte, la plus jouée d’abord', () => {
  const grid = buildMapGrid(soloEmprise())

  it('colonnes : nom, matchs, bilan V / D / autres ; la colonne de repli sans nom porte son nombre de cartes', () => {
    expect(grid.columns.map((c) => [c.key, c.name, c.matches, c.wins, c.losses, c.others, c.otherMaps, c.filmed])).toEqual([
      ['aq', 'Carte Alpha', 2, 1, 1, 0, 0, 2],
      ['rc', 'Carte Bravo', 1, 1, 0, 0, 0, 1],
      ['others', '', 1, 0, 0, 1, 3, 0],
    ])
  })

  it('synthèse des bonus par carte ; colonne sans film « sans film »', () => {
    const bonus = grid.sections.find((s) => s.resource === 'powerup')!
    expect(bonus.summary!.cells.map((c) => (c.kind === 'value' ? `${c.us}–${c.them}` : c.kind))).toEqual(['90–70', '21–17', 'nofilm'])
  })

  it('armes spéciales « non classé » sur une carte filmée sans niveaux de socle ; frags de la feuille partout', () => {
    const power = grid.sections.find((s) => s.resource === 'power_weapon')!
    expect(power.summary!.cells.map((c) => c.kind)).toEqual(['value', 'untiered', 'nofilm'])
    expect(power.kills!.cells.map((c) => (c.kind === 'value' ? `${c.us}–${c.them}` : c.kind))).toEqual(['11–10', '5–5', '2–1'])
  })

  it('« qui chez moi » : les parts de l’objet sur la carte', () => {
    const camo = grid.sections.find((s) => s.resource === 'powerup')!.items.find((r) => r.object?.key === 'powerup_camo')!
    const c = camo.cells[0]
    expect(c.kind === 'value' && c.who).toEqual([
      { xuid: ME, taken: 15 },
      { xuid: null, taken: 35 },
    ])
  })

  it('carte filmée sans camp connu : « camp inconnu », jamais « rien à prendre »', () => {
    const b = soloEmprise()
    b.maps = [{ ...b.maps![1], matches_measured: 0 }]
    const bonus = buildMapGrid(b).sections.find((s) => s.resource === 'powerup')!
    expect(bonus.summary!.cells[0]).toEqual({ kind: 'noteam' })
  })
  it('sans carte (Halo 5) : aucune colonne', () => {
    expect(buildMapGrid(soloEmpriseSansFilm()).columns).toEqual([])
  })
})

describe('buildMinePickups — Mes prises dans mon camp', () => {
  const mine = buildMinePickups(soloEmprise(), nameOf)!

  it('groupes par ressource, les râteliers repliés', () => {
    expect(mine.groups.map((g) => [g.resource, g.folded])).toEqual([
      ['powerup', false],
      ['power_weapon', false],
      ['rack', true],
    ])
  })

  it('objets pris par mon camp triés par volume ; moi / reste du camp ; échelle = le plus gros objet', () => {
    const power = mine.groups[1].rows
    expect(power.map((r) => [r.object.key, r.me, r.rest, r.camp])).toEqual([
      ['spnkr', 30, 99, 129],
      ['sniper', 0, 100, 100],
    ])
    expect(mine.max).toBe(129)
  })

  it('bonus perdus des deux camps (gardés + lâchés)', () => {
    expect(mine.losses).toEqual({ us: { lost: 12, taken: 111 }, them: { lost: 7, taken: 87 } })
  })

  it('sans prise : pas de modèle (la carte se retire)', () => {
    expect(buildMinePickups(soloEmpriseSansFilm(), nameOf)).toBeNull()
  })
})

describe('buildEquipmentRows — Équipement pris, et ce que j’en ai fait', () => {
  const rows = buildEquipmentRows(soloEmprise())

  it('ordre du Go ; mesurées : servi / gardé / lâché pour moi et pour le reste de mon camp', () => {
    expect(rows.map((r) => r.family)).toEqual(['grapple', 'wall', 'sensor', 'shroud_screen', 'thruster'])
    expect(rows[1]).toEqual({ family: 'wall', measured: true, me: [52, 0, 32], rest: [146, 7, 151], takenMe: 23 })
    expect(rows[2]).toMatchObject({ me: [6, 1, 54], rest: [16, 4, 183] })
  })

  it('non mesurées : mes lâchers seulement', () => {
    expect(rows[0]).toEqual({ family: 'grapple', measured: false, droppedMe: 84 })
    expect(rows[4]).toEqual({ family: 'thruster', measured: false, droppedMe: 65 })
  })

  it('une famille que personne n’a tenue dans le lobby est retirée, même quand d’autres ont des comptes', () => {
    expect(rows.map((r) => r.family)).not.toContain('repair_field')
  })

  it('une famille tenue par le seul adversaire est gardée, mes comptes et ceux de mon camp à zéro', () => {
    expect(rows.find((r) => r.family === 'shroud_screen')).toEqual({ family: 'shroud_screen', measured: true, me: [0, 0, 0], rest: [0, 0, 0], takenMe: 0 })
  })

  it('non mesurées : gardées sur les lâchers du lobby, même sans lâcher de ma part ; retirées sans aucun', () => {
    const b = soloEmprise()
    const eq = b.equipment!
    eq.families = (eq.families ?? []).map((f) =>
      f.family === 'grapple' ? { ...f, dropped_me: 0 } : f.family === 'thruster' ? { ...f, dropped_me: 0, dropped_lobby: 0 } : f,
    )
    const fams = buildEquipmentRows(b).map((r) => r.family)
    expect(fams).toContain('grapple')
    expect(fams).not.toContain('thruster')
  })

  it('sans film : aucune ligne ; rien de tenu dans le lobby : aucune ligne', () => {
    expect(buildEquipmentRows(soloEmpriseSansFilm())).toEqual([])
    const vide = soloEmprise()
    vide.equipment = {
      matches_measured: 1,
      families: [
        {
          family: 'wall', measured: true, me: { taken: 0, used: 0, kept: 0, dropped: 0 }, rest: { taken: 0, used: 0, kept: 0, dropped: 0 },
          lobby: { taken: 0, used: 0, kept: 0, dropped: 0 },
        },
        { family: 'grapple', measured: false, dropped_me: 0 },
      ],
    }
    expect(buildEquipmentRows(vide)).toEqual([])
  })
})

describe('buildLivesModel — Mes vies : près d’un coéquipier ou seul', () => {
  it('parts des vies et des frags, frags par vie de chaque côté, vies écartées', () => {
    const m = buildLivesModel(lives())!
    expect(m.lives).toBe(1859)
    expect(m.near).toEqual({ lives: 1558, kills: 1250 })
    expect(m.livesNearShare).toBeCloseTo(1558 / 1859)
    expect(m.killsNearShare).toBeCloseTo(1250 / 1512)
    expect(m.perLifeNear).toBeCloseTo(1250 / 1558)
    expect(m.perLifeAlone).toBeCloseTo(262 / 301)
    expect([m.excludedUnlocated, m.excludedNoRadar, m.excludedUnpublishable]).toEqual([59, 0, 0])
  })

  it('vies d’un match au journal des morts non publiable : comptées dans leur propre cause', () => {
    expect(buildLivesModel({ ...lives(), excluded_unpublishable: 12 })!.excludedUnpublishable).toBe(12)
  })

  it('aucun côté seul : frags par vie absents pour lui (pas de division par zéro)', () => {
    const m = buildLivesModel({ ...lives(), alone: { lives: 0, kills: 0 } })!
    expect(m.perLifeAlone).toBeNull()
    expect(m.killsNearShare).toBe(1)
  })

  it('aucune vie rangée (toutes écartées, ou bloc absent) : pas de carte', () => {
    expect(buildLivesModel({ ...lives(), near: { lives: 0, kills: 0 }, alone: { lives: 0, kills: 0 }, excluded_no_radar: 40 })).toBeNull()
    expect(buildLivesModel(undefined)).toBeNull()
  })
})

describe('usagesSections — un seul prédicat pour la page et l’état vide', () => {
  const page = (extra: Partial<TimeseriesPageResponse>) => ({ match_rows: MATCH_ROWS, ...extra }) as TimeseriesPageResponse
  const solo = { ...block2209(), squad: [{ xuid: 'xj', gamertag: 'JGtm' }] }

  it('périmètre filmé complet : les huit blocs', () => {
    const s = usagesSections(page({ emprise: soloEmprise(), lives_near_teammate: lives(), formes_retenues: solo, weapon_range: {} as never }), true, nameOf)
    expect(s).toEqual({ range: true, bilan: true, carte: true, mine: true, prendre: true, lives: true, objectif: true, equipment: true })
  })

  it('Halo 5 sans film : seul « Prendre, et s’en servir » (frags de la feuille de match)', () => {
    const s = usagesSections(page({ emprise: soloEmpriseSansFilm() }), false, nameOf)
    expect(s).toEqual({ range: false, bilan: false, carte: false, mine: false, prendre: true, lives: false, objectif: false, equipment: false })
  })

  it('portée sans la capability : retirée même avec le bloc', () => {
    expect(usagesSections(page({ weapon_range: {} as never }), false, nameOf).range).toBe(false)
  })

  it('rien : tous faux (l’onglet porte son état vide)', () => {
    expect(Object.values(usagesSections(page({}), true, nameOf)).some(Boolean)).toBe(false)
  })

  it('objectif retiré sans match à objectif', () => {
    expect(usagesSections(page({ formes_retenues: { ...solo, matches: [] } }), true, nameOf).objectif).toBe(false)
  })
})

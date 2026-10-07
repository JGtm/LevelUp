/**
 * Tests — matchEmprise.logic : les modèles des cartes de l'Emprise de la Vue match, sur les deux
 * témoins (`matchEmprise.fixtures.ts`) et sur leurs variantes.
 */
import { describe, expect, it } from 'vitest'

import type { MatchEmpriseBlock, SquadEmpriseObject } from '@/lib/api/types'

import { FLOOD_GULCH, STARBOARD, STARBOARD_LIVES, STARBOARD_SCOREBOARD, XUID } from './matchEmprise.fixtures'
import {
  buildMatchControl,
  buildMatchEmpriseModels,
  buildMatchLives,
  buildMatchProduction,
  buildMatchSheets,
  buildMatchYield,
  hasEquipmentEmpriseCard,
  matchCoverage,
  matchEmpriseCards,
} from './matchEmprise.logic'

const nameOf = (o: SquadEmpriseObject) => o.label ?? o.key
const withMatch = (b: MatchEmpriseBlock, over: Partial<NonNullable<MatchEmpriseBlock['matches']>[number]>): MatchEmpriseBlock => ({
  ...b,
  matches: [{ ...b.matches![0], ...over }],
})

describe('buildMatchControl — D, contrôle des ressources, par match', () => {
  it('témoin du 22/09 : piste par ressource puis une piste par objet, dans l’ordre des ressources', () => {
    const c = buildMatchControl(STARBOARD)
    expect(c.rows.map((r) => `${r.object ? '  ' : ''}${r.object?.label ?? r.resource} ${r.us}–${r.them}`)).toEqual([
      'powerup 5–2',
      '  Surbouclier 5–2',
      'power_weapon 0–2',
      '  Épée à énergie 0–2',
      'rack 4–3',
      '  Fusil électrique 2–2',
      '  VK78 Commando 2–0',
      '  CQS48 Bulldog 0–1',
    ])
    expect(c.rows[0].padsEmptied).toBe(10)
    expect(c.racks).toBe(3)
    expect(c.unclassified).toBeNull()
  })

  it('témoin du 24/07 : les prises sur emplacement non identifié (19 / 12), aucune piste de bonus', () => {
    const c = buildMatchControl(FLOOD_GULCH)
    expect(c.unclassified).toEqual({ us: 19, them: 12 })
    expect(c.rows.filter((r) => !r.object).map((r) => `${r.resource} ${r.us}–${r.them}`)).toEqual(['power_weapon 19–11', 'rack 29–40'])
  })

  it('sans film ou sans équipe connue : aucune ligne, et pas de ligne non identifiée', () => {
    for (const over of [{ has_film: false }, { team_known: false }]) {
      const c = buildMatchControl(withMatch(FLOOD_GULCH, over))
      expect(c.rows).toEqual([])
      expect(c.unclassified).toBeNull()
    }
  })

  it('véhicules : seulement quand le match les mesure', () => {
    const vehicle = { resource: 'vehicle', taken: { us: 3, them: 1 }, objects: [] }
    const base = withMatch(STARBOARD, { resources: [...STARBOARD.matches![0].resources!, vehicle] })
    expect(buildMatchControl(base).rows.some((r) => r.resource === 'vehicle')).toBe(false)
    expect(buildMatchControl(withMatch(base, { vehicles: 'measured', resources: base.matches![0].resources })).rows.some((r) => r.resource === 'vehicle')).toBe(true)
  })
})

describe('buildMatchSheets — E, prises par joueur', () => {
  it('une fiche par joueur de l’équipe, dans l’ordre de `players`, SANS fiche du reste', () => {
    const s = buildMatchSheets(STARBOARD, nameOf)!
    expect(s.owners.map((o) => o.gamertag)).toEqual(['JGtm', 'XL JACOB', 'Madina97294', 'Chocoboflor'])
    expect(s.sections.map((x) => x.resource)).toEqual(['powerup', 'rack'])
    const bonus = s.sections[0]
    expect(bonus.lines[0].taken).toEqual([2, 0, 1, 2])
    expect(bonus.totals).toEqual([2, 0, 1, 2])
    const racks = s.sections[1]
    expect(racks.lines.map((l) => [nameOf(l.object), l.taken])).toEqual([
      ['Fusil électrique', [0, 2, 0, 0]],
      ['VK78 Commando', [1, 1, 0, 0]],
    ])
  })

  it('aucune prise de l’équipe : pas de fiches', () => {
    expect(buildMatchSheets(withMatch(STARBOARD, { has_film: false }), nameOf)).toBeNull()
    expect(buildMatchSheets(null, nameOf)).toBeNull()
  })
})

describe('buildMatchProduction — G, raisons fermées', () => {
  it('22/09 : bonus et armes spéciales ont leur ligne, aucune raison', () => {
    const p = buildMatchProduction(STARBOARD)
    expect(p.rows.map((r) => r.resource)).toEqual(['powerup', 'power_weapon'])
    expect(p.pending).toEqual([])
  })

  it('24/07 : temps d’effet mais journal non publiable → frags non mesurés, barre fine gardée', () => {
    const p = buildMatchProduction(FLOOD_GULCH)
    expect(p.pending).toEqual([
      { resource: 'powerup', reason: 'powerupKillsUnpublished', exposure: { kind: 'effect_ms', value: { us: 166700, them: 216900 } } },
    ])
    expect(p.rows.map((r) => r.resource)).toEqual(['power_weapon'])
  })

  it('journal publiable, temps d’effet, aucun frag → « aucun frag pendant l’effet »', () => {
    const p = buildMatchProduction({ ...FLOOD_GULCH, kill_journal: 'publishable' })
    expect(p.pending[0].reason).toBe('powerupNoKills')
  })

  it('lecture du journal indisponible → « lecture indisponible », jamais « non publiable »', () => {
    const p = buildMatchProduction({ ...FLOOD_GULCH, kill_journal: 'unavailable' })
    expect(p.pending[0]).toMatchObject({ resource: 'powerup', reason: 'powerupKillsUnavailable' })
  })

  it('ni temps d’effet ni frag → « aucun temps d’effet mesuré »', () => {
    const p = buildMatchProduction({ ...FLOOD_GULCH, production: FLOOD_GULCH.production!.filter((x) => x.resource !== 'powerup') })
    expect(p.pending[0]).toEqual({ resource: 'powerup', reason: 'powerupNoEffect' })
  })

  it('armes spéciales : feuille illisible → non mesuré ; feuille lue sans frag → 0 frag', () => {
    const sansArmes = { ...STARBOARD, production: STARBOARD.production!.filter((x) => x.resource !== 'power_weapon') }
    expect(buildMatchProduction({ ...sansArmes, sheet_unavailable: 'sheet_load_failed' }).pending).toEqual([{ resource: 'power_weapon', reason: 'sheetFailed' }])
    const zero = { ...STARBOARD, production: [STARBOARD.production![0], { resource: 'power_weapon', kills: { us: 0, them: 0 } }] }
    expect(buildMatchProduction(zero).pending).toEqual([{ resource: 'power_weapon', reason: 'powerZero', exposure: undefined }])
  })

  it('sans film : aucune raison de bonus (rien de mesuré), la feuille reste lue', () => {
    const p = buildMatchProduction(withMatch(STARBOARD, { has_film: false }))
    expect(p.pending.some((x) => x.resource === 'powerup')).toBe(false)
  })

  it('frags aux armes spéciales sans prise mesurée → note « aucune prise… » sous la barre', () => {
    const p = buildMatchProduction({ ...STARBOARD, production: [{ resource: 'power_weapon', kills: { us: 3, them: 1 } }] })
    expect(p.noPickupNote).toEqual(['power_weapon'])
  })

  it('véhicules mesurés par le titre mais pas sur ce match → non mesuré', () => {
    const veh = { ...STARBOARD, vehicles: {} as NonNullable<MatchEmpriseBlock['vehicles']> }
    expect(buildMatchProduction(veh).pending).toContainEqual({ resource: 'vehicle', reason: 'vehicleUnmeasured' })
    expect(buildMatchProduction(STARBOARD).pending.some((x) => x.resource === 'vehicle')).toBe(false)
  })
})

describe('buildMatchYield — H, raisons fermées', () => {
  it('22/09 : l’adversaire sans temps d’effet, l’équipe sans prise d’arme spéciale', () => {
    expect(buildMatchYield(STARBOARD).pending).toEqual([
      { resource: 'powerup', reason: { kind: 'noEffect', team: false, teamEffectMs: 71600, teamKills: 2 } },
      { resource: 'power_weapon', reason: { kind: 'noPickup', team: true, us: 0, them: 2 } },
    ])
  })

  it('24/07 : journal non publiable → bonus non mesuré ; armes spéciales calculées', () => {
    const y = buildMatchYield(FLOOD_GULCH)
    expect(y.pending).toEqual([{ resource: 'powerup', reason: { kind: 'powerupUnpublished' } }])
    expect(y.rows.map((r) => r.resource)).toEqual(['power_weapon'])
  })

  it('lecture du journal indisponible → bonus « lecture indisponible »', () => {
    expect(buildMatchYield({ ...FLOOD_GULCH, kill_journal: 'unavailable' }).pending).toEqual([{ resource: 'powerup', reason: { kind: 'powerupUnavailable' } }])
  })

  it('match non mesuré : aucune raison', () => {
    expect(buildMatchYield(withMatch(STARBOARD, { team_known: false })).pending).toEqual([])
  })
})

describe('buildMatchLives — I, une ligne par joueur dans l’ordre des fiches', () => {
  it('suit l’ordre des fiches, pas celui du repo ; témoin JGtm 11 près / 2 seul, 8 / 0 frags', () => {
    const l = buildMatchLives(STARBOARD_LIVES, STARBOARD.players!)!
    expect(l.rows.map((r) => r.gamertag)).toEqual(['JGtm', 'XL JACOB', 'Madina97294', 'Chocoboflor'])
    expect(l.rows[0].model).toMatchObject({ near: { lives: 11, kills: 8 }, alone: { lives: 2, kills: 0 } })
  })

  it('un joueur sans vie rangée garde sa ligne (modèle nul) ; aucun joueur rangé → null', () => {
    const l = buildMatchLives({ players: STARBOARD_LIVES.players!.filter((p) => p.xuid !== XUID.jacob) }, STARBOARD.players!)!
    expect(l.rows[1]).toEqual({ xuid: XUID.jacob, gamertag: 'XL JACOB', model: null })
    expect(buildMatchLives({ players: [] }, STARBOARD.players!)).toBeNull()
  })

  it('additionne les vies écartées des trois causes', () => {
    const players = STARBOARD_LIVES.players!.map((p, i) => ({ ...p, excluded_unlocated: i, excluded_no_radar: 1, excluded_unpublishable: 2 }))
    const l = buildMatchLives({ players }, STARBOARD.players!)!
    expect([l.excludedUnlocated, l.excludedNoRadar, l.excludedUnpublishable]).toEqual([6, 4, 8])
  })
})

describe('couverture et présence des cartes', () => {
  it('couverture : film décodé et joueurs présents à la fin (partis exclus)', () => {
    const board = [...STARBOARD_SCOREBOARD, { xuid: 'x-p', gamertag: 'Parti', team_side: 't1', left_in_progress: true }] as typeof STARBOARD_SCOREBOARD
    expect(matchCoverage(STARBOARD, board)).toEqual({ filmed: true, present: 8 })
    expect(matchCoverage(null, board)).toBeNull()
  })

  it('22/09 : toutes les cartes de l’Emprise sont présentes', () => {
    const cards = matchEmpriseCards(buildMatchEmpriseModels(STARBOARD, STARBOARD_LIVES, nameOf))
    expect(cards).toEqual({ control: true, sheets: true, production: true, yield: true, lives: true })
    expect(hasEquipmentEmpriseCard(cards)).toBe(true)
  })

  it('sans bloc : aucune carte, et la section ne se pose pas pour elles', () => {
    const cards = matchEmpriseCards(buildMatchEmpriseModels(null, null, nameOf))
    expect(hasEquipmentEmpriseCard(cards)).toBe(false)
  })
})

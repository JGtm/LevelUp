/**
 * usageLogic.test.ts — la logique du bloc « usages d'équipement, socles et
 * objectifs » (S3). Les invariants éprouvés sont ceux de la doctrine §1 :
 * un champ ABSENT du contrat n'est JAMAIS un zéro (nil ≠ 0), les parts et parités
 * se formatent au dixième, la piste du lobby refuse de découper sans frontière
 * nous/eux, la bande de régularité classe chaque match contre la parité.
 */
import { describe, expect, it } from 'vitest'

import type { SessionUsageBlock, SessionUsageMetric } from '@/lib/api/types'

import { USAGE_TEXT } from './usageI18n'
import { buildCadenceGrid, buildObjectiveFamilyGrid, buildSquadRoleGrid } from './usageGrids'
import {
  buildGaugeRow,
  buildLobbyTrack,
  buildRegularityBand,
  equipmentMetrics,
  formatUsageCount,
  formatUsagePct,
  formatUsageRate,
  metricKind,
  teamOfLobbyParityPct,
  usageAvailability,
} from './usageLogic'

const t = USAGE_TEXT.fr

/** Une métrique minimale : les champs requis seuls, le reste ABSENT (nil). */
function metric(overrides: Partial<SessionUsageMetric> & { key: string }): SessionUsageMetric {
  return {
    player_total: 0,
    lobby_total: 0,
    matches_above_lobby_parity: 0,
    ...overrides,
  }
}

describe('formatage des parts, cadences et comptes', () => {
  it('formate une part au dixième, virgule en FR, point en EN', () => {
    expect(formatUsagePct(45.62, 'fr')).toBe('45,6 %')
    expect(formatUsagePct(45.62, 'en')).toBe('45.6%')
  })

  it('nil ≠ 0 : une part absente rend un tiret, une part nulle rend 0,0 %', () => {
    expect(formatUsagePct(null, 'fr')).toBe('—')
    expect(formatUsagePct(undefined, 'fr')).toBe('—')
    expect(formatUsagePct(0, 'fr')).toBe('0,0 %')
  })

  it('formate une cadence à une décimale, tiret quand absente', () => {
    expect(formatUsageRate(1.26, 'fr')).toBe('1,3')
    expect(formatUsageRate(undefined, 'fr')).toBe('—')
  })

  it('formate une durée en m:ss — un 0 MESURÉ s écrit 0:00, jamais un tiret', () => {
    expect(formatUsageCount(125, 'fr', true)).toBe('2:05')
    expect(formatUsageCount(0, 'fr', true)).toBe('0:00')
    expect(formatUsageCount(null, 'fr', true)).toBe('—')
  })
})

describe('parité du dénominateur « mon camp / lobby »', () => {
  it('dérive 100 × équipe / lobby, et refuse un lobby vide ou absent', () => {
    expect(teamOfLobbyParityPct(4, 8)).toBe(50)
    expect(teamOfLobbyParityPct(undefined, 8)).toBeNull()
    expect(teamOfLobbyParityPct(4, 0)).toBeNull()
  })
})

describe('classement et tri des grandeurs', () => {
  it('classe les clés du contrat, ensemble ouvert côté deployed_*', () => {
    expect(metricKind('camo_episodes')).toBe('camo')
    expect(metricKind('deployed_wall')).toBe('wall')
    expect(metricKind('deployed_sensor')).toBe('deployed_other')
    expect(metricKind('pad_pickups')).toBe('pads')
    expect(metricKind('mystere')).toBe('other')
  })

  it('exclut pad_pickups du bloc équipement et ordonne canoniquement', () => {
    const sorted = equipmentMetrics([
      metric({ key: 'pad_pickups' }),
      metric({ key: 'dropped_objects' }),
      metric({ key: 'grapple_pulls' }),
      metric({ key: 'camo_episodes' }),
    ])
    expect(sorted.map((m) => m.key)).toEqual([
      'camo_episodes',
      'grapple_pulls',
      'dropped_objects',
    ])
  })
})

describe('buildGaugeRow — jauges de parts et parités', () => {
  const shares = {
    player_total: 9,
    team_total: 20,
    lobby_total: 43,
    team_share_of_lobby_pct: 45.6,
    player_share_of_team_pct: 20.5,
    player_share_of_lobby_pct: 9.3,
  }

  it('rend les trois dénominateurs du §7 avec leurs textes d honnêteté', () => {
    const row = buildGaugeRow({
      key: 'pads',
      label: 'Prises de socle',
      shares,
      teamParityPct: 25,
      lobbyParityPct: 12.5,
      teamOfLobbyParityPct: 50,
      rangeMinPct: 5,
      rangeMaxPct: 40,
      t,
      locale: 'fr',
    })
    const [camp, joueurEquipe, joueurLobby] = row.gauges
    expect(camp.valueText).toBe('45,6 %')
    expect(camp.honestyText).toBe('20 sur 43')
    expect(joueurEquipe.valueText).toBe('20,5 %')
    expect(joueurEquipe.parityPct).toBe(25)
    // L étendue n existe QUE sur la jauge joueur/équipe.
    expect(joueurEquipe.rangeMinPct).toBe(5)
    expect(joueurEquipe.rangeMaxPct).toBe(40)
    expect(camp.rangeMinPct).toBeNull()
    expect(joueurLobby.honestyText).toBe('9 sur 43')
  })

  it('nil ≠ 0 : un scope à camp inconnu rend des jauges vides, jamais 0 %', () => {
    const row = buildGaugeRow({
      key: 'x',
      label: 'X',
      shares: { player_total: 3, lobby_total: 12, player_share_of_lobby_pct: 25 },
      teamParityPct: null,
      lobbyParityPct: 12.5,
      teamOfLobbyParityPct: null,
      t,
      locale: 'fr',
    })
    const [camp, joueurEquipe, joueurLobby] = row.gauges
    expect(camp.valuePct).toBeNull()
    expect(camp.valueText).toBe('—')
    expect(joueurEquipe.valuePct).toBeNull()
    expect(joueurEquipe.honestyText).toBe('3 sur —')
    expect(joueurLobby.valuePct).toBe(25)
  })
})

describe('buildLobbyTrack — piste du lobby découpée par joueur', () => {
  const squadPlayers = [
    { xuid: '1', gamertag: 'Madina97294' },
    { xuid: '2', gamertag: 'Chocoboflor' },
  ]

  it('découpe moi + coéquipiers + reste d équipe + eux (hachuré)', () => {
    const track = buildLobbyTrack({
      meLabel: 'JGtm',
      shares: { player_total: 9, team_total: 20, lobby_total: 43 },
      squadPlayers,
      squadShares: [
        { xuid: '1', total: 6 },
        { xuid: '2', total: 3 },
      ],
      t,
      locale: 'fr',
    })
    expect(track).not.toBeNull()
    expect(track!.map((s) => [s.kind, s.count])).toEqual([
      ['me', 9],
      ['squad', 6],
      ['squad', 3],
      ['team-rest', 2],
      ['enemy', 23],
    ])
    // Le segment adverse n a NI nom d équipe NI couleur : son libellé est anonyme.
    expect(track!.at(-1)!.label).toBe(t.segEnemy)
  })

  it('refuse de découper sans frontière nous/eux (team_total absent)', () => {
    const track = buildLobbyTrack({
      meLabel: 'JGtm',
      shares: { player_total: 9, lobby_total: 43 },
      squadPlayers: [],
      squadShares: null,
      t,
      locale: 'fr',
    })
    expect(track).toBeNull()
  })

  it('borne les résidus à zéro (scopes joueur/équipe légèrement disjoints)', () => {
    const track = buildLobbyTrack({
      meLabel: 'JGtm',
      shares: { player_total: 25, team_total: 20, lobby_total: 22 },
      squadPlayers: [],
      squadShares: null,
      t,
      locale: 'fr',
    })
    expect(track!.every((s) => s.count >= 0)).toBe(true)
  })
})

describe('buildRegularityBand — une case par match mesuré', () => {
  it('teinte chaque match par son écart à la parité, tiret pour le non-mesuré', () => {
    const cells = buildRegularityBand(
      [
        { match_id: 'a', player_share_of_team_pct: 40 },
        { match_id: 'b', player_share_of_team_pct: 25.5 },
        { match_id: 'c', player_share_of_team_pct: 10 },
        { match_id: 'd' },
      ],
      25,
      t,
      'fr',
    )
    expect(cells.map((c) => c.tone)).toEqual(['above', 'near', 'below', 'unmeasured'])
    expect(cells[3].tooltip).toBe(t.bandTipUnmeasured(4))
  })

  it('une part de 0 % MESURÉE est « sous la parité », pas « non mesurée »', () => {
    const cells = buildRegularityBand([{ match_id: 'a', player_share_of_team_pct: 0 }], 25, t, 'fr')
    expect(cells[0].tone).toBe('below')
  })

  it('sans parité de session, tout est non mesuré', () => {
    const cells = buildRegularityBand(
      [{ match_id: 'a', player_share_of_team_pct: 40 }],
      undefined,
      t,
      'fr',
    )
    expect(cells[0].tone).toBe('unmeasured')
  })
})

describe('buildCadenceGrid — grille alignée des cadences', () => {
  const inks = {
    columnColor: () => 'var(--ink)',
    rowAccent: () => undefined,
  }

  it('solo : moi + les deux agrégats, un filet entre les deux groupes', () => {
    const grid = buildCadenceGrid({
      metrics: [metric({ key: 'camo_episodes', player_per_10min: 0.8, lobby_per_10min: 5.2 })],
      squadPlayers: [],
      meLabel: 'JGtm',
      t,
      locale: 'fr',
      ...inks,
    })
    expect(grid!.rows.map((r) => r.key)).toEqual(['me', 'team', 'lobby'])
    expect(grid!.separators).toEqual([1])
  })

  it('escouade : une ligne par coéquipier suivi, alignée par xuid', () => {
    const grid = buildCadenceGrid({
      metrics: [
        metric({
          key: 'camo_episodes',
          player_per_10min: 0.8,
          squad: [{ xuid: '1', total: 4, per_10min: 0.5 }],
        }),
      ],
      squadPlayers: [{ xuid: '1', gamertag: 'Madina97294' }],
      meLabel: 'JGtm',
      t,
      locale: 'fr',
      ...inks,
    })
    expect(grid!.rows.map((r) => r.key)).toEqual(['me', 'squad-1', 'team', 'lobby'])
    expect(grid!.cells[1][0].text).toBe('0,5')
  })

  it('nil ≠ 0 : une cadence absente rend une cellule non mesurée, pas un zéro', () => {
    const grid = buildCadenceGrid({
      metrics: [metric({ key: 'grapple_pulls', player_per_10min: 1.2 })],
      squadPlayers: [],
      meLabel: 'JGtm',
      t,
      locale: 'fr',
      ...inks,
    })
    const teamCell = grid!.cells[1][0]
    expect(teamCell.value).toBeNull()
    expect(teamCell.text).toBe('—')
    expect(teamCell.fraction).toBe(0)
  })

  it('sans grandeur, pas de grille', () => {
    expect(
      buildCadenceGrid({ metrics: [], squadPlayers: [], meLabel: 'x', t, locale: 'fr', ...inks }),
    ).toBeNull()
  })
})

describe('grilles d objectifs', () => {
  const inks = { columnColor: () => 'var(--ink)', rowAccent: () => undefined }

  it('famille sans un rôle → cellule non mesurée, colonnes ordonnées', () => {
    const grid = buildObjectiveFamilyGrid({
      families: [
        {
          family: 'ctf',
          matches: 2,
          roles: [
            { role: 'defend', player_total: 3, lobby_total: 20, player_share_of_team_pct: 38.1 },
            { role: 'take', player_total: 1, lobby_total: 9, player_share_of_team_pct: 11.5 },
          ],
        },
        {
          family: 'oddball',
          matches: 1,
          roles: [
            {
              role: 'hold',
              is_duration: true,
              player_total: 61,
              lobby_total: 700,
              player_share_of_team_pct: 15.5,
            },
          ],
        },
      ],
      t,
      locale: 'fr',
      ...inks,
    })
    expect(grid!.columns.map((c) => c.key)).toEqual(['take', 'defend', 'hold'])
    // ctf n a pas de « tenir » : cellule non mesurée, jamais un 0.
    expect(grid!.cells[0][2].value).toBeNull()
    expect(grid!.cells[0][2].text).toBe('—')
    // oddball « tenir » : l infobulle écrit la durée en m:ss (1:01).
    expect(grid!.cells[1][2].tooltip).toContain('1:01')
  })

  it('grille escouade absente en solo', () => {
    expect(
      buildSquadRoleGrid({
        roles: [{ role: 'take', player_total: 1, lobby_total: 9 }],
        squadPlayers: [],
        meLabel: 'x',
        t,
        locale: 'fr',
        ...inks,
      }),
    ).toBeNull()
  })
})

describe('usageAvailability — états du bloc', () => {
  const base: SessionUsageBlock = {
    available: true,
    matches_measured: 8,
    matches_total: 9,
  }

  it('absent du payload → caché (pas de bloc fantôme)', () => {
    expect(usageAvailability(undefined, t)).toEqual({ kind: 'hidden' })
  })

  // LES DEUX RAISONS DE N'AVOIR RIEN NE SE DISENT PAS PAREIL (2026-09-05, registre L4).
  it("titre sans résumé d'usage (unsupported) → caché, pas de carte morte", () => {
    expect(
      usageAvailability({ ...base, available: false, unavailable_reason: 'unsupported' }, t),
    ).toEqual({ kind: 'hidden' })
  })

  it('lecture échouée (load_failed) → état vide AVEC la raison (transitoire, il faut le dire)', () => {
    expect(
      usageAvailability({ ...base, available: false, unavailable_reason: 'load_failed' }, t),
    ).toEqual({ kind: 'empty', message: t.unavailableLoadFailed })
  })

  it('0 match mesuré → état vide « aucun film », bloc présent', () => {
    expect(usageAvailability({ ...base, matches_measured: 0 }, t)).toEqual({
      kind: 'empty',
      message: t.unavailableNoMeasured,
    })
  })

  it('disponible et mesuré → ok', () => {
    expect(usageAvailability(base, t)).toEqual({ kind: 'ok' })
  })
})

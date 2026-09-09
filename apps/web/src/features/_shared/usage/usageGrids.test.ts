/**
 * usageGrids.test.ts — les grilles alignées (cadences, familles d'objectif, escouade) du
 * bloc « usages d'équipement, armes spéciales et objectifs » (S3). Extrait de
 * `usageLogic.test.ts` le 2026-09-09 (étape E5.1bis, scission de taille — CLAUDE.md n°5) au
 * moment du déménagement du bloc vers `features/_shared/usage/`.
 */
import { describe, expect, it } from 'vitest'

import type { SessionUsageMetric } from '@/lib/api/types'

import { USAGE_TEXT } from './usageI18n'
import { buildCadenceGrid, buildObjectiveFamilyGrid, buildSquadRoleGrid } from './usageGrids'

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

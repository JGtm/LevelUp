/**
 * Jeu de données de la vue Escouade : une réponse `view: 'squad'` à trois membres (le joueur
 * principal en premier), avec les indicateurs des groupes `squad` et `members`.
 */
import type { TrendsIndicator, TrendsPageResponse, TrendsPoint } from '@/lib/api/types'

import { dailyPoints, horizonCells, indicator, response } from './tendances.fixture'

export const MEMBERS = ['JGtm', 'Alice', 'Bob'] as const

/** Les membres de la réponse Escouade (xuid = `x-` + gamertag en minuscules), joueur principal en premier. */
export const SQUAD_MEMBERS = MEMBERS.map((gamertag) => ({
  gamertag,
  xuid: `x-${gamertag.toLowerCase()}`,
}))

/** Part de frags de l'escouade de chaque membre sur un point : 0,5 / 0,3 / 0,2. */
const SHARES: Record<string, number> = { JGtm: 0.5, Alice: 0.3, Bob: 0.2 }

function squadRow(
  key: string,
  group: 'squad' | 'members',
  options: {
    variant?: string
    unit?: string
    decimals?: number
    points?: TrendsPoint[]
    prev?: number
    inMatrix?: boolean
  } = {},
): TrendsIndicator {
  const points = options.points ?? dailyPoints(5)
  return indicator({
    key,
    group,
    variant: options.variant,
    unit: options.unit ?? 'number',
    decimals: options.decimals ?? 2,
    in_matrix: options.inMatrix ?? true,
    horizons: horizonCells({ 90: { prev_value: options.prev } }),
    series: { match: points, day: points, week: points, month: points },
  })
}

/** Les indicateurs de la vue Escouade, au complet. */
export function squadIndicators(): TrendsIndicator[] {
  const ratio = { unit: 'ratio', decimals: 3 }
  return [
    squadRow('win_rate', 'squad', { ...ratio, prev: 0.5 }),
    squadRow('win_rate_alone', 'squad', { ...ratio, prev: 0.4 }),
    squadRow('match_count', 'squad', { decimals: 0, prev: 3 }),
    squadRow('squad_share_of_team_kills', 'squad', { ...ratio, prev: 0.55 }),
    squadRow('mmr_gap', 'squad', { prev: 4 }),
    ...MEMBERS.map((gt) => squadRow('kda', 'members', { variant: gt, prev: 2 })),
    ...MEMBERS.map((gt) =>
      squadRow('member_share_of_squad_kills', 'members', {
        variant: gt,
        ...ratio,
        points: dailyPoints(5).map((p) => ({ ...p, value: SHARES[gt] })),
      }),
    ),
  ]
}

/** Une réponse de la vue Escouade ; `overrides` remplace des champs. */
export function squadResponse(overrides: Partial<TrendsPageResponse> = {}): TrendsPageResponse {
  return response({
    view: 'squad',
    game_types: [{ key: 'ranked_slayer', matches: 30 }],
    indicators: squadIndicators(),
    members: SQUAD_MEMBERS,
    ...overrides,
  })
}

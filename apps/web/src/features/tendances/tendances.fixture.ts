/**
 * Jeu de données des tests de l'onglet Tendances : une réponse d'API minimale mais complète
 * (12 mois, 4 horizons), construite par petites fonctions pour que chaque test pose
 * seulement ce qu'il vérifie.
 */
import type {
  TrendsHorizonCell,
  TrendsIndicator,
  TrendsMonthCell,
  TrendsPageResponse,
  TrendsPoint,
} from '@/lib/api/types'

export const MONTHS = [
  '2025-10',
  '2025-11',
  '2025-12',
  '2026-01',
  '2026-02',
  '2026-03',
  '2026-04',
  '2026-05',
  '2026-06',
  '2026-07',
  '2026-08',
  '2026-09',
]

/** Douze cellules de mois : `values[i]` = valeur du mois i (`undefined` = mois vide). */
export function monthCells(values: (number | undefined)[], z = 0.5): TrendsMonthCell[] {
  return MONTHS.map((_, i) => {
    const value = values[i]
    return value === undefined
      ? { matches: 0 }
      : { value, matches: 20, z }
  })
}

/** Les quatre horizons (365, 90, 30, 7) ; `overrides` remplace une cellule par jours. */
export function horizonCells(
  overrides: Partial<Record<number, Partial<TrendsHorizonCell>>> = {},
): TrendsHorizonCell[] {
  return [365, 90, 30, 7].map((days) => ({
    days,
    value: 0.5,
    matches: 40,
    prev_value: 0.45,
    prev_matches: 38,
    z: 1,
    ...overrides[days],
  }))
}

export function indicator(overrides: Partial<TrendsIndicator> = {}): TrendsIndicator {
  return {
    key: 'win_rate',
    group: 'results',
    unit: 'ratio',
    decimals: 3,
    better: 1,
    in_matrix: true,
    months: monthCells(MONTHS.map(() => 0.5)),
    horizons: horizonCells(),
    series: { match: null, day: null, week: null, month: null },
    ...overrides,
  }
}

export function response(overrides: Partial<TrendsPageResponse> = {}): TrendsPageResponse {
  return {
    as_of: '2026-10-05T12:00:00Z',
    view: 'solo',
    game_type: '',
    timezone: 'Europe/Paris',
    game_types: [
      { key: 'ranked_slayer', matches: 120 },
      { key: 'arena_slayer', matches: 80 },
      { key: 'chaine_inconnue', matches: 5 },
    ],
    capabilities: { mmr: true, csr: true, lusr: true, objectives: true, equipment: true },
    months: MONTHS,
    indicators: [indicator()],
    members: [],
    calendar: [],
    win_loss: [],
    medals: [],
    mix: { day: [], week: [], month: [] },
    ...overrides,
  }
}

const DAY_MS = 24 * 60 * 60 * 1000

/** `n` points journaliers, le dernier à `as_of` : la valeur d'un point = `base + i`. */
export function dailyPoints(n: number, base = 1): TrendsPoint[] {
  const fin = new Date('2026-10-05T12:00:00Z').getTime()
  return Array.from({ length: n }, (_, i) => ({
    t: new Date(fin - (n - 1 - i) * DAY_MS).toISOString(),
    value: base + i,
    matches: 3,
  }))
}

/**
 * Un indicateur tracé : la même suite de `n` points à tous les pas, hors « match » pour
 * `win_rate`. `prev` renseigne `prev_value` sur l'horizon 90 j.
 */
export function seriesIndicator(
  key: string,
  options: { variant?: string; n?: number; prev?: number; unit?: string; decimals?: number } = {},
): TrendsIndicator {
  const points = dailyPoints(options.n ?? 5)
  return indicator({
    key,
    variant: options.variant,
    unit: options.unit ?? 'number',
    decimals: options.decimals ?? 2,
    in_matrix: false,
    horizons: horizonCells({ 90: { prev_value: options.prev } }),
    series: { match: points, day: points, week: points, month: points },
  })
}

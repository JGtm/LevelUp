/**
 * Tests des définitions de la vue Escouade : ordre des six graphiques, couleurs par membre,
 * absences au pas « match », dégradation sans MMR, moyenne d'avant, règle des 2 points et
 * barres empilées (pourcentages, découpe par horizon).
 */
import { describe, expect, it } from 'vitest'

import { SQUAD_MAIN_PLAYER_TOKEN, SQUAD_TEAMMATE_COLOR_TOKENS } from '@/features/squad/colors'
import type { TrendsPageResponse } from '@/lib/api/types'

import { dailyPoints } from './tendances.fixture'
import type { Step } from './tendances.logic'
import type { TendancesChartDef } from './tendancesCharts.logic'
import { squadIndicators, squadResponse } from './tendancesSquad.fixture'
import {
  buildTendancesSquadGrids,
  isStackedChart,
  roundSharesToTotal,
  type TendancesAnyChartDef,
  type TendancesStackedChartDef,
} from './tendancesSquadCharts.logic'

const labelOf = (key: string, variant?: string) => (variant ? `${key}|${variant}` : key)

function charts(data: TrendsPageResponse, step: Step = 'day', horizon = 90) {
  const [grille] = buildTendancesSquadGrids({ data, horizon, step, locale: 'fr', labelOf })
  return grille?.charts ?? []
}

const lines = (id: string, list: TendancesAnyChartDef[]) =>
  list.find((c) => c.id === id) as TendancesChartDef

describe('buildTendancesSquadGrids — la grille', () => {
  it('une seule grille sans intertitre, six graphiques dans l’ordre', () => {
    const grilles = buildTendancesSquadGrids({
      data: squadResponse(),
      horizon: 90,
      step: 'day',
      locale: 'fr',
      labelOf,
    })
    expect(grilles).toHaveLength(1)
    expect(grilles[0].id).toBe('squad')
    expect(grilles[0].title).toBe('')
    expect(grilles[0].charts.map((c) => c.title)).toEqual([
      "Taux de victoire avec et sans l'escouade",
      'FDA par membre',
      "Part des frags de l'escouade par membre",
      "Part des frags de l'équipe faite par l'escouade",
      'MMR équipe − adversaires',
      'Matchs joués ensemble',
    ])
  })

  it('taux de victoire : deux courbes nommées, repère 0,5, moyenne d’avant des deux', () => {
    const chart = lines('win-rate-with-alone', charts(squadResponse()))
    expect(chart.input.curves.map((c) => c.name)).toEqual([
      "Avec l'escouade",
      'Sans coéquipier suivi',
    ])
    expect(chart.input.reference).toBe(0.5)
    expect(chart.input.curves.map((c) => c.prevMean)).toEqual([0.5, 0.4])
  })

  it('FDA par membre : une courbe par membre, jetons de joueur dans l’ordre, repère 0', () => {
    const chart = lines('kda-by-member', charts(squadResponse()))
    expect(chart.input.curves.map((c) => c.name)).toEqual(['JGtm', 'Alice', 'Bob'])
    expect(chart.input.curves.map((c) => c.color)).toEqual([
      SQUAD_MAIN_PLAYER_TOKEN,
      SQUAD_TEAMMATE_COLOR_TOKENS[0],
      SQUAD_TEAMMATE_COLOR_TOKENS[1],
    ])
    expect(chart.input.reference).toBe(0)
    expect(chart.input.curves.every((c) => c.prevMean === 2)).toBe(true)
  })

  it('part de l’équipe, écart de MMR (repère 0) et matchs ensemble : une courbe pleine chacun', () => {
    const list = charts(squadResponse())
    expect(lines('squad_share_of_team_kills', list).input.curves[0].prevMean).toBe(0.55)
    const ecart = lines('mmr_gap', list)
    expect(ecart.input.reference).toBe(0)
    expect(ecart.input.curves[0].prevMean).toBe(4)
    expect(lines('match_count', list).input.curves[0].name).toBe('match_count')
  })
})

describe('buildTendancesSquadGrids — absences', () => {
  it('au pas « match » : plus de taux de victoire, de barres ni de matchs ensemble', () => {
    expect(charts(squadResponse(), 'match').map((c) => c.id)).toEqual([
      'kda-by-member',
      'squad_share_of_team_kills',
      'mmr_gap',
    ])
  })

  it('sans capacité MMR : pas d’écart de MMR', () => {
    const data = squadResponse({
      capabilities: { mmr: false, csr: false, lusr: false, objectives: false, equipment: false },
    })
    expect(charts(data).map((c) => c.id)).not.toContain('mmr_gap')
    expect(charts(data)).toHaveLength(5)
  })

  it('un graphique dont toutes les courbes ont moins de 2 points disparaît', () => {
    const data = squadResponse({
      indicators: squadIndicators().map((i) =>
        i.key === 'win_rate_alone' || i.key === 'win_rate'
          ? { ...i, series: { ...i.series, day: dailyPoints(1) } }
          : i,
      ),
    })
    const ids = charts(data).map((c) => c.id)
    expect(ids).not.toContain('win-rate-with-alone')
    expect(ids).toContain('kda-by-member')
  })

  it('une seule des deux courbes de taux de victoire reste : le graphique est tracé avec elle', () => {
    const data = squadResponse({
      indicators: squadIndicators().map((i) =>
        i.key === 'win_rate' ? { ...i, series: { ...i.series, day: null } } : i,
      ),
    })
    const chart = lines('win-rate-with-alone', charts(data))
    expect(chart.input.curves.map((c) => c.name)).toEqual(['Sans coéquipier suivi'])
  })

  it('rien à tracer : aucune grille', () => {
    const data = squadResponse({ indicators: [] })
    expect(
      buildTendancesSquadGrids({ data, horizon: 90, step: 'day', locale: 'fr', labelOf }),
    ).toEqual([])
  })

  it('un point hors de ]as_of − horizon, as_of] ne compte pas', () => {
    // Cinq points journaliers (as_of, −1 j … −4 j) : à 3 j, seuls as_of, −1 j et −2 j sont dedans.
    const kda = lines('kda-by-member', charts(squadResponse(), 'day', 3))
    expect(kda.input.curves[0].points).toHaveLength(3)
  })
})

describe('buildTendancesSquadGrids — barres empilées', () => {
  const stacked = (data = squadResponse(), step: Step = 'day', horizon = 90) =>
    charts(data, step, horizon).find((c) => isStackedChart(c)) as
      | TendancesStackedChartDef
      | undefined

  it('un bâton par intervalle du pas, un composant par membre, en pourcentage entier', () => {
    const chart = stacked()!
    const points = chart.stacked.series[0].datapoints
    expect(points).toHaveLength(5)
    expect(points[0].components).toEqual({ JGtm: 50, Alice: 30, Bob: 20 })
    expect(chart.stacked.componentOrder).toEqual(['JGtm', 'Alice', 'Bob'])
  })

  it('mêmes couleurs que la courbe de FDA', () => {
    expect(stacked()!.stacked.componentColors).toEqual({
      JGtm: SQUAD_MAIN_PLAYER_TOKEN,
      Alice: SQUAD_TEAMMATE_COLOR_TOKENS[0],
      Bob: SQUAD_TEAMMATE_COLOR_TOKENS[1],
    })
  })

  it('découpe par horizon : seuls les intervalles dont le début est dans l’horizon', () => {
    expect(stacked(squadResponse(), 'day', 3)?.stacked.series[0].datapoints).toHaveLength(3)
  })

  it('absent au pas « match », et quand aucun intervalle n’est dans l’horizon', () => {
    expect(stacked(squadResponse(), 'match')).toBeUndefined()
    const loin = squadResponse({
      indicators: squadIndicators().map((i) =>
        i.key === 'member_share_of_squad_kills'
          ? {
              ...i,
              series: {
                ...i.series,
                day: [{ t: '2024-01-01T00:00:00Z', value: 0.5, matches: 2 }],
              },
            }
          : i,
      ),
    })
    expect(stacked(loin)).toBeUndefined()
  })

  it('un membre sans point sur un bâton n’y figure pas', () => {
    const data = squadResponse({
      indicators: squadIndicators().map((i) =>
        i.key === 'member_share_of_squad_kills' && i.variant === 'Bob'
          ? {
              ...i,
              series: {
                ...i.series,
                day: dailyPoints(5)
                  .slice(0, 2)
                  .map((p) => ({ ...p, value: 0.2 })),
              },
            }
          : i,
      ),
    })
    const points = stacked(data)!.stacked.series[0].datapoints
    expect(points).toHaveLength(5)
    expect(Object.keys(points[4].components)).toEqual(['JGtm', 'Alice'])
  })
})

describe('roundSharesToTotal — arrondi au plus fort reste', () => {
  const total = (values: number[]) => values.reduce((a, b) => a + b, 0)

  it('trois tiers : le total vaut 100, l’unité manquante va au premier', () => {
    const r = roundSharesToTotal([1 / 3, 1 / 3, 1 / 3])
    expect(total(r)).toBe(100)
    expect(r).toEqual([34, 33, 33])
  })

  it('0,335 / 0,335 / 0,33 : le total vaut 100', () => {
    const r = roundSharesToTotal([0.335, 0.335, 0.33])
    expect(total(r)).toBe(100)
    expect(r).toEqual([34, 33, 33])
  })

  it('un seul membre à 1 : 100', () => {
    expect(roundSharesToTotal([1])).toEqual([100])
  })

  it('parts dont la somme vaut 0,5 : le total vaut 50', () => {
    const r = roundSharesToTotal([0.25, 0.125, 0.125])
    expect(total(r)).toBe(50)
  })
})

describe('buildTendancesSquadGrids — barres empilées, règle des deux points', () => {
  const withDays = (n: number) =>
    squadResponse({
      indicators: squadIndicators().map((i) =>
        i.key === 'member_share_of_squad_kills'
          ? { ...i, series: { ...i.series, day: dailyPoints(n).map((p) => ({ ...p, value: 0.5 })) } }
          : i,
      ),
    })
  const stacked = (data: TrendsPageResponse) => charts(data).find((c) => isStackedChart(c))

  it('un seul intervalle dans l’horizon : pas de graphique', () => {
    expect(stacked(withDays(1))).toBeUndefined()
  })

  it('deux intervalles : le graphique existe', () => {
    expect(stacked(withDays(2))).toBeDefined()
  })

  it('un bâton de trois tiers totalise 100', () => {
    const tiers = squadResponse({
      indicators: squadIndicators().map((i) =>
        i.key === 'member_share_of_squad_kills'
          ? { ...i, series: { ...i.series, day: dailyPoints(3).map((p) => ({ ...p, value: 1 / 3 })) } }
          : i,
      ),
    })
    const chart = stacked(tiers) as TendancesStackedChartDef
    for (const bar of chart.stacked.series[0].datapoints) {
      expect(Object.values(bar.components).reduce((a, b) => a + b, 0)).toBe(100)
    }
  })
})

/**
 * Tests des définitions de la section « Évolution » : contenu et ordre des deux grilles, règle
 * des 2 points, taux de victoire au pas « match », dégradation sans MMR, une courbe par variante,
 * moyenne de la période d'avant.
 */
import { describe, expect, it } from 'vitest'

import type { TrendsIndicator, TrendsPageResponse } from '@/lib/api/types'

import { response, seriesIndicator } from './tendances.fixture'
import type { Step } from './tendances.logic'
import { buildTendancesGrids } from './tendancesCharts.logic'

const labelOf = (key: string, variant?: string) => (variant ? `${key}|${variant}` : key)

const VERSUS_KEYS = [
  'kda',
  'kills_per_match',
  'deaths_per_match',
  'win_rate',
  'performance_score',
  'accuracy',
  'avg_life_seconds',
  'offensive_conversion',
  'defensive_resistance',
  'enemy_mmr',
]

function fullIndicators(): TrendsIndicator[] {
  return [
    ...VERSUS_KEYS.map((k) => seriesIndicator(k)),
    seriesIndicator('csr_value', { variant: 'Slayer classé', prev: 1.5 }),
    seriesIndicator('csr_value', { variant: 'Objectif classé', prev: 1.5 }),
    seriesIndicator('lusr_value', { variant: 'arena_slayer', prev: 2 }),
    seriesIndicator('lusr_value', { variant: 'btb', prev: 2 }),
    seriesIndicator('team_mmr', { prev: 2 }),
    seriesIndicator('avg_damage_dealt', { prev: 2 }),
    seriesIndicator('avg_damage_taken', { prev: 2 }),
    seriesIndicator('objective_take_share', { prev: 0.3 }),
    seriesIndicator('objective_defend_share', { prev: 0.3 }),
    seriesIndicator('objective_hold_share', { prev: 0.3 }),
    seriesIndicator('objective_parity', { prev: 0.3 }),
  ]
}

function grids(
  data: TrendsPageResponse,
  step: Step = 'day',
  horizon = 90,
) {
  return buildTendancesGrids({ data, horizon, step, locale: 'fr', labelOf })
}

describe('buildTendancesGrids — grille « Face au MMR adverse »', () => {
  it('huit graphiques dans l’ordre, titres « … face au MMR adverse », MMR sur l’axe de droite', () => {
    const [grille] = grids(response({ indicators: fullIndicators() }))
    expect(grille.id).toBe('versus-mmr')
    expect(grille.title).toBe('Face au MMR adverse')
    expect(grille.charts.map((c) => c.id)).toEqual([
      'kda',
      'kills_per_match',
      'deaths_per_match',
      'win_rate',
      'performance_score',
      'accuracy',
      'avg_life_seconds',
      'yield-resistance',
    ])
    expect(grille.charts[0].title).toBe('kda face au MMR adverse')
    expect(grille.charts[7].title).toBe('Rendement et résistance face au MMR adverse')
    for (const c of grille.charts) expect(c.input.right?.name).toBe('enemy_mmr')
  })

  it('repères, jetons de couleur et courbes jumelées', () => {
    const [grille] = grids(response({ indicators: fullIndicators() }))
    const byId = Object.fromEntries(grille.charts.map((c) => [c.id, c.input]))
    expect(byId.kda.reference).toBe(0)
    expect(byId.win_rate.reference).toBe(0.5)
    expect(byId.performance_score.reference).toBe(50)
    expect(byId['yield-resistance'].reference).toBe(1)
    expect(byId.accuracy.reference).toBeUndefined()
    expect(byId.kills_per_match.curves[0].color).toBe('stat-kills')
    expect(byId.deaths_per_match.curves[0].color).toBe('stat-deaths')
    expect(byId.kda.curves[0].color).toBe('chart-series-1')
    expect(byId['yield-resistance'].curves.map((c) => c.color)).toEqual([
      'chart-series-1',
      'chart-series-2',
    ])
  })

  it('aucune moyenne de la période d’avant dans cette grille', () => {
    const indicators = VERSUS_KEYS.map((k) => seriesIndicator(k, { prev: 1 }))
    const [grille] = grids(response({ indicators }))
    for (const c of grille.charts) {
      for (const courbe of c.input.curves) expect(courbe.prevMean).toBeUndefined()
    }
  })

  it('win_rate absent au pas « match »', () => {
    const [grille] = grids(response({ indicators: fullIndicators() }), 'match')
    expect(grille.charts.map((c) => c.id)).not.toContain('win_rate')
    expect(grille.charts).toHaveLength(7)
  })

  it('sans capacité MMR : titre de grille et de graphiques sans MMR, un seul axe', () => {
    const data = response({
      indicators: fullIndicators(),
      capabilities: { mmr: false, csr: true, lusr: true, objectives: true, equipment: true },
    })
    const [grille] = grids(data)
    expect(grille.title).toBe('Résultats et combat')
    expect(grille.charts[0].title).toBe('kda')
    for (const c of grille.charts) expect(c.input.right).toBeUndefined()
  })

  it('série enemy_mmr de moins de 2 points sur l’horizon : un seul axe', () => {
    const indicators = fullIndicators().map((i) =>
      i.key === 'enemy_mmr' ? seriesIndicator('enemy_mmr', { n: 1 }) : i,
    )
    const [grille] = grids(response({ indicators }))
    expect(grille.title).toBe('Résultats et combat')
    expect(grille.charts[0].input.right).toBeUndefined()
  })

  it('courbe de moins de 2 points retirée ; sans courbe de gauche, pas de graphique', () => {
    const indicators = fullIndicators().map((i) =>
      i.key === 'accuracy' ? seriesIndicator('accuracy', { n: 1 }) : i,
    )
    const [grille] = grids(response({ indicators }))
    expect(grille.charts.map((c) => c.id)).not.toContain('accuracy')
  })

  it('rendement seul quand la résistance manque : le graphique reste, à une courbe', () => {
    const indicators = fullIndicators().filter((i) => i.key !== 'defensive_resistance')
    const [grille] = grids(response({ indicators }))
    expect(grille.charts.find((c) => c.id === 'yield-resistance')?.input.curves).toHaveLength(1)
  })

  it('points hors de l’horizon écartés : 7 j ne garde que la semaine', () => {
    const indicators = [seriesIndicator('kda', { n: 20 })]
    const [grille] = grids(response({ indicators }), 'day', 7)
    expect(grille.charts[0].input.curves[0].points).toHaveLength(7)
  })
})

describe('buildTendancesGrids — grille « Niveau, combat et style »', () => {
  const level = () => grids(response({ indicators: fullIndicators() }))[1]

  it('cinq graphiques dans l’ordre', () => {
    const grille = level()
    expect(grille.id).toBe('level')
    expect(grille.title).toBe('Niveau, combat et style')
    expect(grille.charts.map((c) => c.id)).toEqual(['csr', 'lusr', 'mmr-pair', 'damage', 'objectives'])
    expect(grille.charts.map((c) => c.title)).toEqual([
      'CSR par file classée',
      'LUSR par type de partie (non classé)',
      "MMR des adversaires et de l'équipe",
      'Dégâts infligés et subis par match',
      'Objectifs : prendre, défendre, tenir',
    ])
  })

  it('une courbe par variante : CSR par file, LUSR par chaîne (jeton et nom)', () => {
    const [csr, lusr] = level().charts
    expect(csr.input.curves.map((c) => c.name)).toEqual(['Slayer classé', 'Objectif classé'])
    expect(csr.input.curves.map((c) => c.color)).toEqual(['chart-series-1', 'chart-series-2'])
    expect(lusr.input.curves).toHaveLength(2)
    expect(lusr.input.curves.map((c) => c.color)).toEqual(['compare-a', 'divergent-pos'])
    expect(lusr.input.curves[0].name).not.toBe('arena_slayer')
  })

  it('MMR adversaire en couleur 5 des séries, équipe à la couleur d’ordre', () => {
    const mmr = level().charts[2]
    expect(mmr.input.curves.map((c) => c.name)).toEqual(['enemy_mmr', 'team_mmr'])
    expect(mmr.input.curves[0].color).toBe('chart-series-5')
    expect(mmr.input.curves[1].color).toBe('chart-series-2')
    expect(mmr.info).toBeDefined()
  })

  it('objectifs : jetons de rôle, parité en pointillé à l’encre d’axe, sans moyenne d’avant', () => {
    const objectifs = level().charts[4].input.curves
    expect(objectifs.map((c) => c.color)).toEqual([
      'objective-role-take',
      'objective-role-defend',
      'objective-role-hold',
      'axis-ink',
    ])
    expect(objectifs[3].dashed).toBe(true)
    expect(objectifs[3].prevMean).toBeUndefined()
  })

  it('moyenne d’avant = prev_value de l’horizon courant, sur les courbes pleines', () => {
    const [csr, , , damage] = level().charts
    expect(csr.input.curves.every((c) => c.prevMean === 1.5)).toBe(true)
    expect(damage.input.curves.every((c) => c.prevMean === 2)).toBe(true)
  })

  it('moyenne d’avant : celle de l’horizon courant, absente quand l’horizon n’en a pas', () => {
    const avec = [seriesIndicator('avg_damage_dealt', { prev: 4 })]
    const [g30] = grids(response({ indicators: avec }), 'day', 30)
    expect(g30.charts[0].input.curves[0].prevMean).toBe(0.45) // cellule 30 j de la fixture
    const [g90] = grids(response({ indicators: avec }), 'day', 90)
    expect(g90.charts[0].input.curves[0].prevMean).toBe(4)
    const sans = [seriesIndicator('avg_damage_dealt')]
    const [g90sans] = grids(response({ indicators: sans }), 'day', 90)
    expect(g90sans.charts[0].input.curves[0].prevMean).toBeUndefined()
  })

  it('bulles d’information : CSR, MMR et objectifs seulement', () => {
    const infos = level().charts.map((c) => Boolean(c.info))
    expect(infos).toEqual([true, false, true, false, true])
  })

  it('capacités absentes : les graphiques correspondants disparaissent', () => {
    const data = response({
      indicators: fullIndicators(),
      capabilities: { mmr: false, csr: false, lusr: false, objectives: false, equipment: true },
    })
    const sorties = grids(data)
    expect(sorties.map((g) => g.id)).toEqual(['versus-mmr', 'level'])
    expect(sorties[1].charts.map((c) => c.id)).toEqual(['damage'])
  })
})

describe('buildTendancesGrids — rien à tracer', () => {
  it('aucun indicateur tracé : aucune grille', () => {
    expect(grids(response({ indicators: [] }))).toEqual([])
    expect(grids(response({ indicators: null }))).toEqual([])
  })

  it('fenêtre de l’axe : de as_of − horizon à as_of', () => {
    const [grille] = grids(response({ indicators: [seriesIndicator('kda')] }), 'day', 30)
    const { from, to } = grille.charts[0].input
    expect(to - from).toBe(30 * 86_400_000)
    expect(new Date(to).toISOString()).toBe('2026-10-05T12:00:00.000Z')
  })
})

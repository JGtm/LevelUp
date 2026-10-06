/**
 * Tests des lignes de « Victoires et défaites » et de « Médailles », et de la série des types
 * de partie : table de formats, comparé ou non, « nouvelle », pas « match » ramené à « day »,
 * découpe par horizon, couleurs, ordre.
 */
import { describe, expect, it } from 'vitest'

import type {
  TrendsMedalsBlock,
  TrendsPageResponse,
  TrendsWinLossBlock,
  TrendsWinLossRow,
} from '@/lib/api/types'

import { response } from './tendances.fixture'
import { buildMedalRows, medalsBlock } from './tendancesMedals.logic'
import { buildMixChart, mixStep } from './tendancesMix.logic'
import {
  WIN_LOSS_FORMATS,
  buildWinLossRows,
  winLossBlock,
  winLossFormat,
  winLossShortfall,
} from './tendancesWinLoss.logic'

const labelOf = (key: string) => `L:${key}`

function wlRow(overrides: Partial<TrendsWinLossRow> = {}): TrendsWinLossRow {
  return {
    key: 'kda',
    group: 'stats',
    matches: 42,
    win_mean: 5.678,
    loss_mean: 1.234,
    z_win: 0.8,
    z_loss: -0.6,
    r: 0.347,
    ...overrides,
  }
}

describe('table de formats de « Victoires et défaites »', () => {
  it('chaque indicateur à son unité et ses décimales, les autres à 1 décimale', () => {
    expect(WIN_LOSS_FORMATS.accuracy).toEqual({ unit: 'ratio', decimals: 3 })
    expect(winLossFormat('avg_life_seconds')).toEqual({ unit: 'seconds', decimals: 1 })
    for (const key of ['kda', 'offensive_conversion', 'defensive_resistance']) {
      expect(winLossFormat(key)).toEqual({ unit: 'number', decimals: 2 })
    }
    for (const key of ['damage_balance', 'mmr_gap']) {
      expect(winLossFormat(key)).toEqual({ unit: 'number', decimals: 0 })
    }
    expect(winLossFormat('avg_damage_dealt')).toEqual({ unit: 'number', decimals: 1 })
  })

  it('valeurs écrites près des points : pourcentage à 1 décimale, secondes, 2 décimales, entier', () => {
    const block: TrendsWinLossBlock = {
      days: 90,
      matches: 60,
      required: 20,
      rows: [
        wlRow({ key: 'accuracy', loss_mean: 0.4521, win_mean: 0.5 }),
        wlRow({ key: 'avg_life_seconds', loss_mean: 23.456, win_mean: 30 }),
        wlRow({ key: 'kda', loss_mean: 1.234, win_mean: 5.678 }),
        wlRow({ key: 'damage_balance', loss_mean: -12.6, win_mean: 40.4 }),
        wlRow({ key: 'avg_damage_dealt', loss_mean: 3.14159, win_mean: 4 }),
      ],
    }
    const rows = buildWinLossRows(block, 'fr', labelOf)
    expect(rows.map((r) => [r.aText, r.bText])).toEqual([
      ['45,2 %', '50,0 %'],
      ['23,5 s', '30,0 s'],
      ['1,23', '5,68'],
      ['−13', '40'],
      ['3,1', '4,0'],
    ])
  })
})

describe('buildWinLossRows', () => {
  const block: TrendsWinLossBlock = {
    days: 90,
    matches: 60,
    required: 20,
    rows: [wlRow({ key: 'kda' }), wlRow({ key: 'mmr_gap', r: -0.2, z_win: 1.2, z_loss: -1.1, matches: 55 })],
  }

  it('une ligne par ligne reçue, dans l’ordre : A = défaites à z_loss, B = victoires à z_win', () => {
    const rows = buildWinLossRows(block, 'fr', labelOf)
    expect(rows.map((r) => r.label)).toEqual(['L:kda', 'L:mmr_gap'])
    expect(rows[0]).toMatchObject({ a: -0.6, b: 0.8 })
    expect(rows[1]).toMatchObject({ a: -1.1, b: 1.2 })
  })

  it('infobulle : défaites, victoires, r signé à 2 décimales, nombre de matchs', () => {
    const rows = buildWinLossRows(block, 'fr', labelOf)
    expect(rows[0].tooltip).toEqual([
      'Défaites : 1,23',
      'Victoires : 5,68',
      'r = +0,35',
      '42 matchs',
    ])
    expect(rows[1].tooltip[2]).toBe('r = −0,20')
    expect(rows[1].tooltip[3]).toBe('55 matchs')
  })

  it('bloc absent ou sans ligne : aucune ligne', () => {
    expect(buildWinLossRows(undefined, 'fr', labelOf)).toEqual([])
    expect(buildWinLossRows({ ...block, rows: [] }, 'fr', labelOf)).toEqual([])
    expect(buildWinLossRows({ ...block, rows: null }, 'fr', labelOf)).toEqual([])
  })

  it('le bloc est celui de l’horizon ; état vide : matchs et seuil du bloc', () => {
    const data = response({
      win_loss: [
        { days: 7, matches: 4, required: 10, rows: [] },
        { days: 90, matches: 60, required: 20, rows: [wlRow()] },
      ],
    })
    expect(winLossBlock(data, 90)?.matches).toBe(60)
    expect(winLossBlock(data, 30)).toBeUndefined()
    expect(winLossShortfall(winLossBlock(data, 7))).toEqual({ n: 4, required: 10 })
    expect(winLossShortfall(undefined)).toEqual({ n: 0, required: 10 })
  })
})

describe('buildMedalRows', () => {
  const names = { current: '90 derniers jours', previous: "90 jours d'avant" }
  const compared: TrendsMedalsBlock = {
    days: 90,
    compared: true,
    rows: [
      { medal_id: 1, name: 'Tueur', rate: 0.5, prev_rate: 0.4 },
      { medal_id: 2, name: 'Sniper', rate: 0.3, prev_rate: 0.4 },
      { medal_id: 3, name: 'Nouvelle', rate: 0.2 },
    ],
  }

  it('comparé : B = taux de l’horizon, A = taux d’avant, étiquette du point B à 2 décimales', () => {
    const rows = buildMedalRows(compared, 'fr', names)
    expect(rows[0]).toMatchObject({ label: 'Tueur', a: 0.4, b: 0.5, bText: '0,50' })
    expect(rows[0].aText).toBeUndefined()
  })

  it('infobulle comparée : taux de l’horizon, taux d’avant, variation en pourcentage', () => {
    const rows = buildMedalRows(compared, 'fr', names)
    expect(rows[0].tooltip).toEqual([
      '90 derniers jours : 0,50 par match',
      "90 jours d'avant : 0,40 par match",
      'Variation : +25 %',
    ])
    expect(rows[1].tooltip[2]).toBe('Variation : −25 %')
  })

  it('taux d’avant nul : « nouvelle », point d’avant à 0', () => {
    const rows = buildMedalRows(compared, 'fr', names)
    expect(rows[2].a).toBe(0)
    expect(rows[2].tooltip[2]).toBe('Variation : nouvelle')
    const zero = buildMedalRows(
      { days: 90, compared: true, rows: [{ medal_id: 4, name: 'Z', rate: 0.1, prev_rate: 0 }] },
      'fr',
      names,
    )
    expect(zero[0].tooltip[2]).toBe('Variation : nouvelle')
  })

  it('non comparé : point B seul, infobulle réduite au taux de l’horizon', () => {
    const rows = buildMedalRows({ ...compared, compared: false }, 'fr', names)
    expect(rows.every((r) => r.a === null)).toBe(true)
    expect(rows[0].b).toBe(0.5)
    expect(rows[0].tooltip).toEqual(['90 derniers jours : 0,50 par match'])
  })

  it('bloc absent ou sans ligne : aucune ligne ; le bloc est celui de l’horizon', () => {
    expect(buildMedalRows(undefined, 'fr', names)).toEqual([])
    expect(buildMedalRows({ days: 7, compared: false, rows: null }, 'fr', names)).toEqual([])
    const data = response({ medals: [compared, { days: 30, compared: false, rows: [] }] })
    expect(medalsBlock(data, 90)).toBe(compared)
    expect(medalsBlock(data, 365)).toBeUndefined()
  })
})

describe('buildMixChart', () => {
  const mix: TrendsPageResponse['mix'] = {
    day: [
      { t: '2026-09-27T22:00:00Z', counts: { ranked_slayer: 9 } }, // avant l’horizon de 7 j
      { t: '2026-09-28T22:00:00Z', counts: { ranked_slayer: 3, arena_slayer: 2 } },
      { t: '2026-10-04T22:00:00Z', counts: { arena_slayer: 4, nouveau_type: 1 } },
    ],
    week: [{ t: '2026-09-21T22:00:00Z', counts: { ranked_slayer: 7 } }],
    month: [
      { t: '2026-07-31T22:00:00Z', counts: { ranked_slayer: 5 } },
      { t: '2026-08-31T22:00:00Z', counts: { ranked_slayer: 6 } },
      { t: '2026-09-30T22:00:00Z', counts: { ranked_slayer: 8 } },
    ],
  }
  const data = response({ mix })
  const base = { data, horizon: 7, step: 'day' as const, locale: 'fr' as const, labelOf }

  it('le pas « match » se ramène à « day »', () => {
    expect(mixStep('match')).toBe('day')
    expect(mixStep('week')).toBe('week')
    expect(buildMixChart({ ...base, step: 'match' })).toEqual(buildMixChart({ ...base, step: 'day' }))
  })

  it('découpe par horizon : seuls les intervalles dont le début est dans l’horizon', () => {
    const chart = buildMixChart(base)
    const points = chart.series[0].datapoints
    expect(points).toHaveLength(2)
    expect(points[0].category).toMatch(/^29 sept/)
    expect(points[1].category).toMatch(/^5 oct/)
    expect(points[0].components).toEqual({ 'L:ranked_slayer': 3, 'L:arena_slayer': 2 })
    expect(points[1].components['L:nouveau_type']).toBe(1)
  })

  it('pas « mois » : catégorie en mois court seul, trois mois sur 90 j', () => {
    const points = buildMixChart({ ...base, horizon: 90, step: 'month' }).series[0].datapoints
    expect(points.map((p) => p.category)).toEqual(['août', 'sept.', 'oct.'])
  })

  it('couleurs : jeton LUSR quand la chaîne y figure, sinon chart-series selon la position', () => {
    const chart = buildMixChart(base)
    expect(chart.componentColors).toEqual({
      'L:ranked_slayer': 'chart-series-1',
      'L:arena_slayer': 'compare-a',
      'L:chaine_inconnue': 'chart-series-3',
      'L:nouveau_type': 'chart-series-4',
    })
  })

  it('ordre : celui de game_types, puis les clés vues dans le mélange', () => {
    expect(buildMixChart(base).componentOrder).toEqual([
      'L:ranked_slayer',
      'L:arena_slayer',
      'L:chaine_inconnue',
      'L:nouveau_type',
    ])
  })

  it('aucun intervalle dans l’horizon : série vide (le wrapper dit « vide »)', () => {
    expect(buildMixChart({ ...base, horizon: 7, step: 'week' }).series).toEqual([])
    expect(buildMixChart({ ...base, data: response() }).series).toEqual([])
  })
})

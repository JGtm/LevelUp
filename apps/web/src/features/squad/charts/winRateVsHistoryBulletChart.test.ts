/**
 * winRateVsHistoryBulletChart.test.ts — bullet chart teammates.02.
 *
 * tokenCssVar stubbed pour retourner le nom du token comme valeur de couleur.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildWinRateVsHistoryBulletOption } from './winRateVsHistoryBulletChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { MapBreakdownRow } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
  resolveToken: (token: string) => `color:${token}`,
}))

function row(
  mapUI: string,
  winRate: number,
  historicalWinRate?: number,
  matchCount = 10,
  historicalMatchCount?: number,
): MapBreakdownRow {
  return {
    map_ui: mapUI,
    match_count: matchCount,
    win_rate: winRate,
    historical_win_rate: historicalWinRate,
    historical_match_count: historicalMatchCount,
  }
}

const OPTS = {
  mapLabelOf: (m: string) => m.toUpperCase(),
  sessionLabel: 'Session',
  historyLabel: 'Historique',
  parityLabel: 'Parité 50 %',
  zeroWinrateLabel: '0 % (toutes défaites)',
}

function makeSeries(rows: MapBreakdownRow[]): ChartSeries<MapBreakdownRow>[] {
  return [{ key: 'win-rate-vs-history-bullet', datapoints: rows }]
}

type SessionDatum = { value: number; itemStyle?: { color: string } }
type HistDatum = { value: number | null; itemStyle?: { color: string; opacity?: number } }
type SeriesShape = {
  name: string
  data: Array<SessionDatum | HistDatum>
  barGap?: string
  markLine?: { data: Array<{ xAxis: number }> }
  markPoint?: { data: Array<{ xAxis: number; yAxis: number; name: string }> }
}

function findByName(opt: ReturnType<typeof buildWinRateVsHistoryBulletOption>, name: string): SeriesShape {
  const series = opt.series as SeriesShape[]
  const found = series.find((s) => s.name === name)
  if (!found) throw new Error(`série "${name}" introuvable`)
  return found
}

describe('buildWinRateVsHistoryBulletOption', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('série vide → option minimale', () => {
    const opt = buildWinRateVsHistoryBulletOption([], OPTS)
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('série sans datapoints → option minimale', () => {
    const opt = buildWinRateVsHistoryBulletOption([{ key: 'k', datapoints: [] }], OPTS)
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('applique mapLabelOf sur les noms de cartes', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('aquarius', 0.5, 0.5)]), OPTS)
    const yAxis = opt.yAxis as { data: string[] }
    expect(yAxis.data[0]).toContain('AQUARIUS')
  })

  it('suffixe (n) = nombre de parties de la session sur le libellé d\'axe', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('aquarius', 0.6, 0.5, 7)]), OPTS)
    const yAxis = opt.yAxis as { data: string[] }
    expect(yAxis.data).toEqual(['AQUARIUS (7)'])
  })

  it('respecte l\'ordre reçu du backend, aucun re-tri par match_count — I12', () => {
    // Le backend (computeMapBreakdown) trie déjà par première apparition
    // chronologique ; le front ne doit PAS re-trier par match_count desc
    // (ancien comportement masquant l'ordre chronologique).
    const rows = [
      row('b', 0.8, 0.5, 5),
      row('a', 0.3, 0.5, 20),
      row('c', 0.6, 0.5, 10),
    ]
    const opt = buildWinRateVsHistoryBulletOption(makeSeries(rows), OPTS)
    const yAxis = opt.yAxis as { data: string[] }
    expect(yAxis.data).toEqual(['B (5)', 'A (20)', 'C (10)'])
  })

  it('tooltip formatter → titre carte + ligne counts session/historique', () => {
    const countsLabel = vi.fn(
      (s: number, h?: number) => `Session : ${s} parties · Historique : ${h === undefined ? '—' : `${h} parties`}`,
    )
    const opt = buildWinRateVsHistoryBulletOption(
      makeSeries([row('aquarius', 0.6, 0.5, 7, 40)]),
      { ...OPTS, countsLabel },
    )
    const formatter = (opt.tooltip as { formatter: (p: unknown) => string }).formatter
    const html = formatter([
      { dataIndex: 0, marker: '', seriesName: 'Historique', value: 50 },
      { dataIndex: 0, marker: '', seriesName: 'Session', value: 60 },
    ])
    expect(countsLabel).toHaveBeenCalledWith(7, 40)
    expect(html).toContain('AQUARIUS')
    expect(html).toContain('Session : 7 parties · Historique : 40 parties')
  })

  it('tooltip counts : historique absent → passe undefined à countsLabel', () => {
    const countsLabel = vi.fn((s: number, h?: number) => `${s}/${h ?? 'na'}`)
    const opt = buildWinRateVsHistoryBulletOption(
      makeSeries([row('x', 0.6, undefined, 3)]),
      { ...OPTS, countsLabel },
    )
    const formatter = (opt.tooltip as { formatter: (p: unknown) => string }).formatter
    formatter([{ dataIndex: 0, marker: '', seriesName: 'Session', value: 60 }])
    expect(countsLabel).toHaveBeenCalledWith(3, undefined)
  })

  it('yAxis inverse activé (carte la plus jouée en haut)', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.5, 0.5)]), OPTS)
    const yAxis = opt.yAxis as { inverse: boolean }
    expect(yAxis.inverse).toBe(true)
  })

  it('session au-dessus du seuil → couleur divergent-pos', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.8, 0.5)]), OPTS)
    const sessionDatum = findByName(opt, 'Session').data[0] as SessionDatum
    expect(sessionDatum.itemStyle?.color).toBe('color:divergent-pos')
  })

  it('session en dessous du seuil → couleur divergent-neg', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.3, 0.6)]), OPTS)
    const sessionDatum = findByName(opt, 'Session').data[0] as SessionDatum
    expect(sessionDatum.itemStyle?.color).toBe('color:divergent-neg')
  })

  it('session dans le seuil → couleur divergent-neutral', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.52, 0.5)]), OPTS)
    const sessionDatum = findByName(opt, 'Session').data[0] as SessionDatum
    expect(sessionDatum.itemStyle?.color).toBe('color:divergent-neutral')
  })

  it('sans historique → couleur divergent-neutral', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.7)]), OPTS)
    const sessionDatum = findByName(opt, 'Session').data[0] as SessionDatum
    expect(sessionDatum.itemStyle?.color).toBe('color:divergent-neutral')
  })

  it('historique → couleur chart-series-1 avec opacité', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.5, 0.6)]), OPTS)
    const histDatum = findByName(opt, 'Historique').data[0] as HistDatum
    expect(histDatum.itemStyle?.color).toBe('color:chart-series-1')
    expect(histDatum.itemStyle?.opacity).toBe(0.85)
  })

  it('valeurs converties en pourcentage (×100, 1 décimale)', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.625, 0.5)]), OPTS)
    const sessionDatum = findByName(opt, 'Session').data[0] as SessionDatum
    expect(sessionDatum.value).toBe(62.5)
  })

  it('barGap=-100% sur les deux séries → overlay (bullet)', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.5, 0.5)]), OPTS)
    expect(findByName(opt, 'Historique').barGap).toBe('-100%')
    expect(findByName(opt, 'Session').barGap).toBe('-100%')
  })

  it('markLine de parité à 50% sur la série historique', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.5, 0.5)]), OPTS)
    const markLine = findByName(opt, 'Historique').markLine
    expect(markLine?.data?.[0]?.xAxis).toBe(50)
  })

  it('markPoint pour les cartes 0% (toutes défaites)', () => {
    const rows = [
      row('a', 0.5, 0.5, 10),
      row('b', 0, 0.5, 8),
      row('c', 0.7, 0.5, 5),
    ]
    const opt = buildWinRateVsHistoryBulletOption(makeSeries(rows), OPTS)
    const markPoint = findByName(opt, 'Session').markPoint
    expect(markPoint?.data).toHaveLength(1)
    expect(markPoint?.data?.[0]).toMatchObject({ xAxis: 0, name: '0 % (toutes défaites)' })
  })

  it('pas de markPoint quand aucune carte 0%', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.5, 0.5)]), OPTS)
    expect(findByName(opt, 'Session').markPoint).toBeUndefined()
  })

  it('xAxis en pourcentage avec formatter {value}%', () => {
    const opt = buildWinRateVsHistoryBulletOption(makeSeries([row('x', 0.5, 0.5)]), OPTS)
    const xAxis = opt.xAxis as { axisLabel: { formatter: string }; min: number; max: number }
    expect(xAxis.axisLabel.formatter).toBe('{value}%')
    expect(xAxis.min).toBe(0)
    expect(xAxis.max).toBe(100)
  })
})

/**
 * Formats des infobulles des Séries temporelles (retours du 2026-10-17) : aucun nombre brut, une
 * ligne par série qui a une valeur, des libellés qui disent ce que la valeur mesure.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { CorrelationDataPair, DistributionBucket, TimeseriesMatchRow } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { buildFdaGapCumulativeOption } from './TimeseriesFdaGapTrend'
import { TimeseriesDistributionHistogram } from './TimeseriesDistributionHistogram'
import { TimeseriesKdaDensity } from './TimeseriesKdaDensity'
import { TimeseriesKdaTrend } from './TimeseriesKdaTrend'
import { buildTimeseriesNetLivesOption } from './TimeseriesNetLivesTrend'
import { TimeseriesScatterWithTrend } from './TimeseriesScatterWithTrend'

let captured: Record<string, unknown> | null = null
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: Record<string, unknown> }) => {
    captured = option
    return <div data-testid="echarts-mock" />
  },
}))

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

type Formatter = (params: unknown) => string
const tip = (opt: unknown): Formatter => (opt as { tooltip: { formatter: Formatter } }).tooltip.formatter
const flat = (html: string) => html.normalize('NFKC')

async function optionOf(ui: React.ReactElement) {
  captured = null
  render(ui)
  await screen.findByTestId('echarts-mock')
  return captured as unknown as Record<string, unknown>
}

describe('infobulles des Séries temporelles : nombres à la langue', () => {
  it('frags, bonus, morts : le bonus (assistances / 3) garde ses décimales, la pile se lit de haut en bas', async () => {
    const rows = [{ match_id: 'm', kills: 7, deaths: 3, assists: 4 }] as TimeseriesMatchRow[]
    const opt = await optionOf(
      <TimeseriesKdaTrend
        rows={rows}
        labels={{ kills: 'Frags', deaths: 'Morts', yAxis: 'y', bonus: 'Bonus', bonusInfo: 'i' }}
      />,
    )
    const html = flat(
      tip(opt)([
        { seriesName: 'Frags', seriesIndex: 0, value: 7, color: '#111', axisValueLabel: '#1' },
        { seriesName: 'Bonus', seriesIndex: 1, value: 4 / 3, color: '#222', axisValueLabel: '#1' },
        { seriesName: 'Morts', seriesIndex: 2, value: 3, color: '#333', axisValueLabel: '#1' },
      ]),
    )
    expect(html).toContain('1,33')
    expect(html).not.toContain('1.3333')
    expect(html.indexOf('Bonus')).toBeLessThan(html.indexOf('Frags'))
  })

  it('histogramme : la bulle nomme la plage (l’axe n’écrit que la borne basse) et compte des matchs', async () => {
    const buckets = [{ bucket_lower: 0.5, bucket_upper: 1.25, count: 4 }] as DistributionBucket[]
    const opt = await optionOf(
      <TimeseriesDistributionHistogram
        buckets={buckets}
        colorToken="chart-series-1"
        xAxisLabel="FDA"
        medianLabel="Médiane"
      />,
    )
    expect((opt.xAxis as { data: string[] }).data).toEqual(['0,5'])
    const html = tip(opt)([{ dataIndex: 0, seriesName: 'Matchs', value: 4, color: '#111', axisValueLabel: '0,5' }])
    expect(html).toContain('0,5–1,25')
    expect(html).toContain('Matchs')
    expect(html).not.toContain('FDA')
  })

  it('densité du FDA : l’abscisse continue s’écrit à deux décimales, la densité en pourcentage', async () => {
    const buckets = [{ bucket_lower: 1, bucket_upper: 2, count: 3 }] as DistributionBucket[]
    const opt = await optionOf(
      <TimeseriesKdaDensity
        buckets={buckets}
        rows={[{ match_id: 'a', kda: 1.2345678 }] as TimeseriesMatchRow[]}
        labels={{ density: 'Densité', rug: 'Matchs', xAxis: 'FDA', mean: 'Moyenne', median: 'Médiane' }}
      />,
    )
    const html = flat(
      tip(opt)([{ seriesName: 'Densité', seriesIndex: 0, value: [1.5, 100], color: '#111', axisValue: 1.2345678 }]),
    )
    expect(html).toContain('FDA 1,23')
    expect(html).toContain('100,0 %')
    const markers = (opt.series as Array<{ markLine?: { data: Array<{ label: { formatter: string } }> } }>)[0].markLine!.data
    expect(markers.map((d) => d.label.formatter)).toEqual(['Moyenne 1,23', 'Médiane 1,23'])
  })

  it('nuage de points : la bulle dit quelle grandeur est quelle (pas « 12.3456, 4 »)', async () => {
    const pts = [{ metric_x_key: 'a', metric_y_key: 'b', x_value: 12.3456, y_value: 4, outcome: 2 }] as CorrelationDataPair[]
    const opt = await optionOf(
      <TimeseriesScatterWithTrend
        points={pts}
        metricXKey="a"
        metricYKey="b"
        xAxisLabel="Durée de vie"
        yAxisLabel="Frags"
        outcomeLabels={{ win: 'Victoire', loss: 'Défaite', unknown: 'Inconnu' }}
        trendLabel="Tendance"
      />,
    )
    const html = flat(tip(opt)({ seriesName: 'Victoire', value: [12.3456, 4] }))
    expect(html).toContain('Victoire')
    expect(html).toContain('Durée de vie')
    expect(html).toContain('12,35')
    expect(html).toContain('Frags')
    expect(html).not.toContain('12.3456')
  })

  it('écart de FDA cumulé et balance de vies : décimales au plus utiles, signe typographique', () => {
    const rows = [
      { match_id: 'a', kda: 2, kda_expected: 1.5, start_time: '2026-10-01T10:00:00Z' },
      { match_id: 'b', kda: 1, kda_expected: 2.3333333333333335, start_time: '2026-10-02T10:00:00Z' },
    ] as TimeseriesMatchRow[]
    const gap = buildFdaGapCumulativeOption(rows, {
      series: 'Cumul',
      real: 'Réel',
      expected: 'Attendu',
      gap: 'Écart',
      perMatch: '/match',
    } as never)
    const html = flat(tip(gap)([{ dataIndex: 1 }]))
    expect(html).not.toMatch(/\d\.\d{3,}/)
    expect(html).toContain('−')

    const net = buildTimeseriesNetLivesOption(
      [{ header: 'h', label: '#1', value: 4 / 3, cumulative: 4 / 3 }] as never,
      { series: 'Cumul', match: 'Match' },
    )
    expect(flat(tip(net)([{ dataIndex: 0 }]))).toContain('<b>+1,3</b>')
  })
})

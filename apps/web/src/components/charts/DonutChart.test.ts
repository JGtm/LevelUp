/**
 * Tests — buildDonutOption (Phase 3 P3.D).
 */
import { describe, it, expect } from 'vitest'
import { buildDonutOption, type ChartPointDonut } from './DonutChart'
import type { ChartSeries } from './ChartCard'

function makeSeries(points: ChartPointDonut[]): ChartSeries<ChartPointDonut>[] {
  return [
    {
      key: 'donut.test',
      meta: { gamertag: 'test' },
      datapoints: points,
    },
  ]
}

interface OptionShape {
  series?: Array<{
    type?: string
    radius?: [string, string]
    data?: Array<{ name: string; value: number; itemStyle?: { color?: string } }>
    label?: { show?: boolean; formatter?: string | ((p: { name?: string; percent?: number }) => string) }
  }>
  legend?: { data?: string[] }
  backgroundColor?: string
  tooltip?: { formatter?: (p: { name?: string; value?: number; percent?: number }) => string }
}

describe('buildDonutOption', () => {
  it('retourne option vide si aucune série', () => {
    const opt = buildDonutOption([]) as OptionShape
    expect(opt.series).toBeUndefined()
    expect(opt.backgroundColor).toBeDefined()
  })

  it('retourne option vide si série sans datapoints', () => {
    const opt = buildDonutOption(makeSeries([])) as OptionShape
    expect(opt.series).toBeUndefined()
  })

  it('génère une série pie avec les slices dans l\'ordre', () => {
    const opt = buildDonutOption(
      makeSeries([
        { name: 'win', value: 8 },
        { name: 'loss', value: 3 },
      ]),
    ) as OptionShape
    expect(opt.series?.[0].type).toBe('pie')
    expect(opt.series?.[0].data).toHaveLength(2)
    expect(opt.series?.[0].data?.[0].name).toBe('win')
    expect(opt.series?.[0].data?.[0].value).toBe(8)
  })

  it('rayon par défaut [50%, 75%]', () => {
    const opt = buildDonutOption(makeSeries([{ name: 'a', value: 1 }])) as OptionShape
    expect(opt.series?.[0].radius).toEqual(['50%', '75%'])
  })

  it('respecte innerRadius / outerRadius custom', () => {
    const opt = buildDonutOption(makeSeries([{ name: 'a', value: 1 }]), {
      innerRadius: '0%',
      outerRadius: '90%',
    }) as OptionShape
    expect(opt.series?.[0].radius).toEqual(['0%', '90%'])
  })

  it('peuple la legend avec les noms de slices', () => {
    const opt = buildDonutOption(
      makeSeries([
        { name: 'win', value: 1 },
        { name: 'loss', value: 1 },
        { name: 'tie', value: 1 },
      ]),
    ) as OptionShape
    expect(opt.legend?.data).toEqual(['win', 'loss', 'tie'])
  })

  it('cache les pourcentages quand showPercent=false', () => {
    const opt = buildDonutOption(makeSeries([{ name: 'a', value: 1 }]), {
      showPercent: false,
    }) as OptionShape
    expect(opt.series?.[0].label?.show).toBe(false)
    expect(opt.series?.[0].label?.formatter).toBe('{b}')
  })

  it('affiche les pourcentages par défaut', () => {
    const opt = buildDonutOption(makeSeries([{ name: 'a', value: 1 }])) as OptionShape
    expect(opt.series?.[0].label?.show).toBe(true)
    // Le nom puis la part à la langue (« 33 % »), jamais le gabarit brut `{d}%` d'ECharts (« 33.33% »).
    const fmt = opt.series?.[0].label?.formatter as (p: { name?: string; percent?: number }) => string
    expect(fmt({ name: 'a', percent: 100 / 3 })).toBe('a\n33 %')
  })

  it('chaque slice a un itemStyle.color résolu', () => {
    const opt = buildDonutOption(
      makeSeries([
        { name: 'win', value: 1 },
        { name: 'loss', value: 1 },
      ]),
    ) as OptionShape
    expect(opt.series?.[0].data?.[0].itemStyle?.color).toBeDefined()
    expect(opt.series?.[0].data?.[1].itemStyle?.color).toBeDefined()
  })

  it('étiquette d’arc par défaut : le nom et le pourcentage', () => {
    const opt = buildDonutOption(makeSeries([{ name: 'a', value: 1 }])) as OptionShape
    const fmt = opt.series?.[0].label?.formatter as (p: { name?: string; percent?: number }) => string
    expect(fmt({ name: 'a', percent: 50 })).toBe('a\n50 %')
  })

  it('l’infobulle écrit le compte et la part à la langue de l’application, le nom échappé', () => {
    const opt = buildDonutOption(makeSeries([{ name: 'a<b', value: 1234 }])) as OptionShape
    const html = opt.tooltip!.formatter!({ name: 'a<b', value: 1234, percent: 100 / 3 }).normalize('NFKC')
    expect(html).toBe('a&lt;b : <b>1 234</b> (33,3 %)')
  })
})

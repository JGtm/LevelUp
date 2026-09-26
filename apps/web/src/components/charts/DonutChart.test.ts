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
    label?: { show?: boolean; formatter?: string }
  }>
  legend?: { data?: string[] }
  backgroundColor?: string
  tooltip?: { formatter?: string }
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
    expect(opt.series?.[0].label?.formatter).toBe('{b}\n{d}%')
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

  // ── arcLabelKind='value' (PLAN_EQUIPEMENT_GACHIS_2026-09-09, décision P11 : « les
  // valeurs sont sur les arcs, la légende ne dit que la couleur »). ──────────────────
  describe('arcLabelKind="value" (P11)', () => {
    it('affiche un label meme si showPercent=false (defaut inchange pour "percent")', () => {
      const opt = buildDonutOption(makeSeries([{ name: 'a', value: 12 }]), {
        showPercent: false,
        arcLabelKind: 'value',
      }) as OptionShape
      expect(opt.series?.[0].label?.show).toBe(true)
    })

    it('le formatter ecrit le nom puis le compte brut (valueLabel si fourni, sinon value)', () => {
      const opt = buildDonutOption(
        makeSeries([{ name: 'Moi', value: 312, valueLabel: '312' }]),
        { arcLabelKind: 'value' },
      ) as unknown as { series: Array<{ label: { formatter: (p: unknown) => string } }> }
      const text = opt.series[0].label.formatter({ name: 'Moi', value: 312, data: { valueLabel: '312' } })
      expect(text).toBe('Moi\n312')
    })

    it('sans arcLabelKind (defaut "percent"), le comportement existant ne change pas', () => {
      const opt = buildDonutOption(makeSeries([{ name: 'a', value: 1 }])) as OptionShape
      expect(opt.series?.[0].label?.formatter).toBe('{b}\n{d}%')
    })
  })
})

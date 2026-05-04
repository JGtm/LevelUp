/**
 * squadPerMinuteChart.test.ts — teammates.14.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildSquadPerMinuteOption } from './squadPerMinuteChart'
import { hexComplement } from '@/lib/accessibility/hexComplement'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerMinuteEntry } from '@/lib/api/types'

vi.mock('@/lib/accessibility', async () => {
  // hexComplement est une fonction pure — on la passe en real impl dans le mock
  // pour que le chart l'utilise sans avoir besoin de DOM ou de palette active.
  const { hexComplement: hc } = await import('@/lib/accessibility/hexComplement')
  return {
    tokenCssVar: (token: string) => `color:${token}`,
    resolveToken: (token: string) => `hex(${token})`,
    hexComplement: hc,
  }
})

// Couleurs non-grises pour que hexComplement retourne un résultat distinct.
const PLAYER_COLOR = '#818cf8' // indigo — couleur positive du joueur "Me"

const OPTS = {
  colorByPlayer: { Me: PLAYER_COLOR, F1: '#00dc82', F2: '#f59e0b' },
  metricLabels: { frags: 'Frags/min', deaths: 'Morts/min', assists: 'Assists/min' },
  perMinuteSuffix: ' /min',
}

function entry(name: string, k: number, d: number, a: number): SquadPerMinuteEntry {
  return { player: name, kills_per_minute: k, deaths_per_minute: d, assists_per_minute: a, match_count: 5 }
}

function makeSeries(rows: SquadPerMinuteEntry[]): ChartSeries<SquadPerMinuteEntry>[] {
  return [{ key: 'k', datapoints: rows }]
}

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadPerMinuteOption', () => {
  it('rows vides → option minimale', () => {
    expect(buildSquadPerMinuteOption([], OPTS)).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('1 série bar par joueur (3 joueurs → 3 series)', () => {
    const opt = buildSquadPerMinuteOption(
      makeSeries([entry('Me', 1.2, 0.7, 0.3), entry('F1', 0.9, 0.5, 0.2), entry('F2', 1.5, 1.0, 0.5)]),
      OPTS,
    )
    const series = opt.series as Array<{ name: string; type: string }>
    expect(series).toHaveLength(3)
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1', 'F2'])
    expect(series.every((s) => s.type === 'bar')).toBe(true)
  })

  it('xAxis = [Frags, Morts, Assists] selon i18n labels', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 1, 1)]), OPTS)
    const xAxis = opt.xAxis as { data: string[] }
    expect(xAxis.data).toEqual(['Frags/min', 'Morts/min', 'Assists/min'])
  })

  it('values : frags positif, morts NÉGATIF (sous l\'axe), assists positif', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1.5, 0.7, 0.3)]), OPTS)
    const series = opt.series as Array<{ data: Array<{ value: number }> }>
    const data = series[0].data
    expect(data[0].value).toBe(1.5)
    expect(data[1].value).toBe(-0.7) // morts négatives
    expect(data[2].value).toBe(0.3)
  })

  it('couleur complémentaire opaque sur morts, couleur joueur sur frags+assists', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 1, 1)]), OPTS)
    const series = opt.series as Array<{ data: Array<{ itemStyle: { color: string; opacity?: number } }> }>
    const data = series[0].data
    const expectedNeg = hexComplement(PLAYER_COLOR)
    expect(data[0].itemStyle.color).toBe(PLAYER_COLOR)      // frags = couleur joueur
    expect(data[0].itemStyle.opacity).toBeUndefined()
    expect(data[1].itemStyle.color).toBe(expectedNeg)        // morts = complémentaire
    expect(data[1].itemStyle.opacity).toBeUndefined()        // opaque (pas d'opacity)
    expect(data[1].itemStyle.color).not.toBe(PLAYER_COLOR)  // must differ from source
    expect(data[2].itemStyle.color).toBe(PLAYER_COLOR)      // assists = couleur joueur
  })

  it('joueur sans couleur mappée → fallback gris #888, complément de #888', () => {
    const opt = buildSquadPerMinuteOption(
      makeSeries([entry('Unknown', 1, 1, 1)]),
      OPTS,
    )
    const series = opt.series as Array<{ data: Array<{ itemStyle: { color: string } }> }>
    expect(series[0].data[0].itemStyle.color).toBe('#888') // frags = gris fallback
    // morts = complément du gris (#888888 → hue 0° → complément = même gris)
    expect(series[0].data[1].itemStyle.color).toBe(hexComplement('#888'))
  })

  it('label formatter retourne valeur ABSOLUE pour les morts (data index 1)', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 0.7, 0.3)]), OPTS)
    const series = opt.series as Array<{
      label: { formatter: (p: { value: number; dataIndex: number }) => string }
    }>
    const fmt = series[0].label.formatter
    expect(fmt({ value: -0.7, dataIndex: 1 })).toBe('0.70')
    expect(fmt({ value: 1.5, dataIndex: 0 })).toBe('1.50')
  })

  it('axe X = ligne blanche en gras (zero line emphasis)', () => {
    const opt = buildSquadPerMinuteOption(makeSeries([entry('Me', 1, 1, 1)]), OPTS)
    const xAxis = opt.xAxis as { axisLine: { lineStyle: { color: string; width: number } } }
    expect(xAxis.axisLine.lineStyle.width).toBe(2)
    expect(xAxis.axisLine.lineStyle.color).toContain('255')
  })
})

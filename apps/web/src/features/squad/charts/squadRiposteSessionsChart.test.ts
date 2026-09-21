/**
 * L'option ECharts de la FRISE de la riposte — la grammaire de référence, tenue.
 *
 * Ce que ces tests cadenassent : un seul axe Y (deux seraient une faute de lecture quand
 * tout est en points de pourcentage), les bâtons à 18 px, la courbe pleine non lissée, le
 * repère d'habituel tireté, et le rang d'étiquettes des volumes sous l'axe des dates.
 */
import { describe, expect, it } from 'vitest'

import type { ChartSeries } from '@/components/charts/ChartCard'
import type { FriseRiposte } from '../squadRiposte.logic'

import { buildSquadRiposteSessionsOption } from './squadRiposteSessionsChart'

const OPTS = {
  aboveLabel: 'au-dessus',
  belowLabel: 'en dessous',
  trendLabel: 'tendance',
  usualLabel: 'habituel 22,1 %',
  yAxisLabel: 'Part des morts ripostées',
  volumeAxisLabel: 'morts mesurées',
  volumeTooltip: (n: number) => `${n} morts mesurées`,
}

function friseDe(over: Partial<FriseRiposte> = {}): FriseRiposte {
  return {
    soirees: [
      { label: '12/09', tauxPct: 24.8, morts: 84, auDessus: true, echantillonFaible: false },
      { label: '14/09', tauxPct: 17.2, morts: 61, auDessus: false, echantillonFaible: false },
    ],
    tendancePct: [24.8, 21],
    habituelPct: 22.1,
    ...over,
  }
}

function option(frise: FriseRiposte) {
  const series: ChartSeries<FriseRiposte>[] = [{ key: 'f', datapoints: [frise] }]
  return buildSquadRiposteSessionsOption(series, OPTS) as Record<string, unknown>
}

describe('buildSquadRiposteSessionsOption', () => {
  it('rend UN SEUL axe Y, en pourcents et ancré à zéro', () => {
    const y = option(friseDe()).yAxis as Record<string, unknown>
    expect(Array.isArray(y)).toBe(false)
    expect(y.min).toBe(0)
    expect((y.axisLabel as { formatter: string }).formatter).toBe('{value} %')
  })

  it('reprend la grammaire de référence : bâtons 18 px, courbe pleine 2 px NON lissée', () => {
    const series = option(friseDe()).series as Record<string, unknown>[]
    const barres = series.filter((s) => s.type === 'bar')
    expect(barres).toHaveLength(2)
    expect(barres.every((s) => s.barMaxWidth === 18)).toBe(true)
    const ligne = series.find((s) => s.type === 'line') as Record<string, unknown>
    expect(ligne.smooth).toBe(false)
    expect((ligne.lineStyle as { width: number }).width).toBe(2)
  })

  it('sépare les soirées en DEUX séries à trous — une entrée de légende par verdict', () => {
    const series = option(friseDe()).series as Record<string, unknown>[]
    const above = series.find((s) => s.name === OPTS.aboveLabel)
    const below = series.find((s) => s.name === OPTS.belowLabel)
    expect(above?.data).toEqual([24.8, null])
    expect(below?.data).toEqual([null, 17.2])
    expect((option(friseDe()).legend as { data: string[] }).data).toEqual([
      OPTS.aboveLabel,
      OPTS.belowLabel,
      OPTS.trendLabel,
    ])
  })

  it('pose le repère d’HABITUEL en ligne TIRETÉE, jamais en bâton', () => {
    const series = option(friseDe()).series as Record<string, unknown>[]
    const markLine = (series[0].markLine ?? {}) as {
      lineStyle: { type: string }
      data: { yAxis: number }[]
    }
    expect(markLine.lineStyle.type).toBe('dashed')
    expect(markLine.data[0].yAxis).toBeCloseTo(22.1, 6)
  })

  it('pose les VOLUMES en second rang d’étiquettes, sans ligne ni graduation', () => {
    const x = option(friseDe()).xAxis as Record<string, unknown>[]
    expect(x).toHaveLength(2)
    expect(x[0].data).toEqual(['12/09', '14/09'])
    expect(x[1].data).toEqual(['84', '61'])
    expect((x[1].axisLine as { show: boolean }).show).toBe(false)
  })

  it('UNE SEULE SOIRÉE rend UN SEUL bâton — c’est déjà un graphe', () => {
    const une = friseDe({
      soirees: [{ label: '20/09', tauxPct: 23.4, morts: 121, auDessus: true, echantillonFaible: false }],
      tendancePct: [23.4],
    })
    const series = option(une).series as Record<string, unknown>[]
    expect(series.find((s) => s.name === OPTS.aboveLabel)?.data).toEqual([23.4])
    expect((option(une).xAxis as { data: string[] }[])[0].data).toEqual(['20/09'])
  })

  it('rend une option VIDE sans soirée : pas d’axes fantômes', () => {
    expect(option(friseDe({ soirees: [], tendancePct: [] })).series).toBeUndefined()
  })
})

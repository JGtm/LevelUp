/**
 * squadFirstEventsChart.test.ts — teammates.17 (butterfly first frag/death).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildSquadFirstEventsOption } from './squadFirstEventsChart'
import type { SquadFirstEvents } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
}))

const COLORS = { Me: '#aaa', F1: '#bbb' }
const OPTS = {
  colorByPlayer: COLORS,
  fragLabel: 'Premier frag',
  deathLabel: 'Première mort',
  matchesSuffix: 'matchs',
}

function data(): SquadFirstEvents {
  return {
    bin_size_seconds: 15,
    bin_labels: ['15s', '30s', '45s', '1m00s'],
    rows: [
      { player: 'Me', kill_counts: [3, 1, 0, 0], death_counts: [0, 1, 2, 1] },
      { player: 'F1', kill_counts: [1, 2, 1, 0], death_counts: [1, 0, 1, 0] },
    ],
  }
}

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadFirstEventsOption', () => {
  it('vide → option minimale', () => {
    expect(buildSquadFirstEventsOption(null, OPTS)).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('aucune row → option minimale', () => {
    expect(buildSquadFirstEventsOption({ bin_size_seconds: 15, bin_labels: ['15s'], rows: [] }, OPTS))
      .toMatchObject({ backgroundColor: 'transparent' })
  })

  it('2 séries par joueur (frag positif + death négatif)', () => {
    const opt = buildSquadFirstEventsOption(data(), OPTS)
    const series = opt.series as Array<{ name: string; data: number[] }>
    expect(series).toHaveLength(4) // 2 joueurs × 2 traces
    expect(series.map((s) => s.name)).toEqual([
      'Me — Premier frag',
      'Me — Première mort',
      'F1 — Premier frag',
      'F1 — Première mort',
    ])
  })

  it('frags positifs, deaths NÉGATIFS (sous l\'axe)', () => {
    const opt = buildSquadFirstEventsOption(data(), OPTS)
    const series = opt.series as Array<{ name: string; data: number[] }>
    const meFrag = series.find((s) => s.name === 'Me — Premier frag')
    const meDeath = series.find((s) => s.name === 'Me — Première mort')
    expect(meFrag?.data).toEqual([3, 1, 0, 0])
    expect(meDeath?.data).toEqual([0, -1, -2, -1])
  })

  it('frag = couleur joueur normale, death = même couleur opacity 0.45', () => {
    const opt = buildSquadFirstEventsOption(data(), OPTS)
    const series = opt.series as Array<{
      name: string
      itemStyle: { color: string; opacity?: number }
    }>
    const meFrag = series.find((s) => s.name === 'Me — Premier frag')
    const meDeath = series.find((s) => s.name === 'Me — Première mort')
    expect(meFrag?.itemStyle).toMatchObject({ color: '#aaa' })
    expect(meFrag?.itemStyle.opacity).toBeUndefined()
    expect(meDeath?.itemStyle).toMatchObject({ color: '#aaa', opacity: 0.45 })
  })

  it('séparateurs verticaux dotted entre chaque bin (markLine sur la 1ère série)', () => {
    const opt = buildSquadFirstEventsOption(data(), OPTS)
    const series = opt.series as Array<{
      name: string
      markLine?: { lineStyle: { type: string }; data: Array<{ xAxis: number }> }
    }>
    const first = series[0]
    expect(first.markLine).toBeDefined()
    expect(first.markLine?.lineStyle.type).toBe('dotted')
    // 4 bins → 3 séparateurs entre.
    expect(first.markLine?.data).toHaveLength(3)
    expect(first.markLine?.data[0].xAxis).toBe(0.5)
  })

  it('xAxis = bin_labels du backend (borne droite "15s", "1m00s"...)', () => {
    const opt = buildSquadFirstEventsOption(data(), OPTS)
    const xAxis = opt.xAxis as { data: string[] }
    expect(xAxis.data).toEqual(['15s', '30s', '45s', '1m00s'])
  })

  it('yAxis labels en valeur ABSOLUE (formatter Math.abs)', () => {
    const opt = buildSquadFirstEventsOption(data(), OPTS)
    const yAxis = opt.yAxis as { axisLabel: { formatter: (v: number) => string } }
    expect(yAxis.axisLabel.formatter(-3)).toBe('3')
    expect(yAxis.axisLabel.formatter(2)).toBe('2')
  })

  it('legend désactivée (pill+combobox sert de légende globale)', () => {
    const opt = buildSquadFirstEventsOption(data(), OPTS)
    const legend = opt.legend as { show: boolean }
    expect(legend.show).toBe(false)
  })

  it('joueur sans couleur mappée → fallback gris #888', () => {
    const d: SquadFirstEvents = {
      bin_size_seconds: 15,
      bin_labels: ['15s'],
      rows: [{ player: 'Unknown', kill_counts: [1], death_counts: [0] }],
    }
    const opt = buildSquadFirstEventsOption(d, { ...OPTS, colorByPlayer: {} })
    const series = opt.series as Array<{ itemStyle: { color: string } }>
    expect(series[0].itemStyle.color).toBe('#888')
  })
})

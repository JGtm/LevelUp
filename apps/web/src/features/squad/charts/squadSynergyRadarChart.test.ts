/**
 * squadSynergyRadarChart.test.ts — teammates.06.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildSquadSynergyRadarOption } from './squadSynergyRadarChart'
import type { SquadSynergyRadarSeries } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
}))

const AXIS_LABELS = {
  combat: 'Combat',
  survival: 'Survie',
  support: 'Support',
  score: 'Score',
  objective: 'Objectif',
  impact: 'Impact',
}

function profile(player: string, values: Partial<Record<string, number>> = {}): SquadSynergyRadarSeries {
  const axes = ['combat', 'survival', 'support', 'score', 'objective', 'impact'].map((k) => ({
    axis: k,
    value: values[k] ?? 50,
    raw: 0,
  }))
  return { player, axes }
}

const COLORS = { Me: '#aaa', F1: '#bbb' }

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadSynergyRadarOption', () => {
  it('vide → option minimale', () => {
    expect(buildSquadSynergyRadarOption([], { colorByPlayer: COLORS, axisLabels: AXIS_LABELS })).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('1 trace par joueur', () => {
    const opt = buildSquadSynergyRadarOption(
      [profile('Me'), profile('F1')],
      { colorByPlayer: COLORS, axisLabels: AXIS_LABELS },
    )
    const series = opt.series as Array<{ data: Array<{ name: string }> }>
    expect(series).toHaveLength(1)
    expect(series[0].data).toHaveLength(2)
    expect(series[0].data.map((d) => d.name)).toEqual(['Me', 'F1'])
  })

  it('couleur appliquée par joueur (cohérence pill/combobox)', () => {
    const opt = buildSquadSynergyRadarOption(
      [profile('Me'), profile('F1')],
      { colorByPlayer: COLORS, axisLabels: AXIS_LABELS },
    )
    const series = opt.series as Array<{ data: Array<{ name: string; itemStyle: { color: string }; lineStyle: { color: string } }> }>
    const me = series[0].data.find((d) => d.name === 'Me')
    const f1 = series[0].data.find((d) => d.name === 'F1')
    expect(me?.itemStyle.color).toBe('#aaa')
    expect(me?.lineStyle.color).toBe('#aaa')
    expect(f1?.itemStyle.color).toBe('#bbb')
  })

  it('PAS d\'aire centrale (areaStyle absent) — multi-profils lisibles', () => {
    const opt = buildSquadSynergyRadarOption(
      [profile('Me'), profile('F1')],
      { colorByPlayer: COLORS, axisLabels: AXIS_LABELS },
    )
    const series = opt.series as Array<{ data: Array<Record<string, unknown>> }>
    expect(series[0].data[0].areaStyle).toBeUndefined()
  })

  it('axisLabels appliqués sur les indicators du radar', () => {
    const opt = buildSquadSynergyRadarOption(
      [profile('Me')],
      { colorByPlayer: COLORS, axisLabels: AXIS_LABELS },
    )
    const radar = opt.radar as { indicator: Array<{ name: string; max: number }> }
    expect(radar.indicator.map((i) => i.name)).toEqual(['Combat', 'Survie', 'Support', 'Score', 'Objectif', 'Impact'])
    expect(radar.indicator.every((i) => i.max === 100)).toBe(true)
  })

  it('joueur sans couleur mappée → fallback gris #888', () => {
    const opt = buildSquadSynergyRadarOption(
      [profile('Unknown')],
      { colorByPlayer: COLORS, axisLabels: AXIS_LABELS },
    )
    const series = opt.series as Array<{ data: Array<{ itemStyle: { color: string } }> }>
    expect(series[0].data[0].itemStyle.color).toBe('#888')
  })
})

import { describe, it, expect, vi } from 'vitest'

import { buildRadarOption, type RadarSeriesPayload } from './RadarChart'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

describe('buildRadarOption', () => {
  const series: RadarSeriesPayload[] = [
    {
      key: 'squad.radar.main',
      meta: { gamertag: 'main', mode_family: 'slayer' },
      axes: [
        { axis: 'combat', value: 80, raw: 200 },
        { axis: 'survival', value: 50, raw: 25 },
        { axis: 'support', value: 30, raw: 24 },
        { axis: 'score', value: 60, raw: 48 },
        { axis: 'objective', value: 0, raw: 0 },
        { axis: 'impact', value: 75, raw: 75 },
      ],
    },
    {
      key: 'squad.radar.f1',
      meta: { gamertag: 'f1' },
      axes: [
        { axis: 'combat', value: 50, raw: 125 },
        { axis: 'survival', value: 40, raw: 20 },
        { axis: 'support', value: 70, raw: 56 },
        { axis: 'score', value: 30, raw: 24 },
        { axis: 'objective', value: 0, raw: 0 },
        { axis: 'impact', value: 60, raw: 60 },
      ],
    },
  ]

  it('extrait les indicateurs depuis le 1er joueur', () => {
    const opt = buildRadarOption(series) as {
      radar: { indicator: { name: string; max: number }[] }
    }
    expect(opt.radar.indicator).toHaveLength(6)
    expect(opt.radar.indicator[0].name).toBe('combat')
    expect(opt.radar.indicator[0].max).toBe(100)
  })

  it('utilise axisLabels pour les noms', () => {
    const opt = buildRadarOption(series, {
      axisLabels: { combat: 'Combat FR', survival: 'Survie' },
    }) as { radar: { indicator: { name: string }[] } }
    expect(opt.radar.indicator[0].name).toBe('Combat FR')
    expect(opt.radar.indicator[1].name).toBe('Survie')
    // Axe sans label → fallback brut
    expect(opt.radar.indicator[2].name).toBe('support')
  })

  it('1 trace par série, valeurs 0..100 alignées', () => {
    const opt = buildRadarOption(series) as {
      series: { data: { name: string; value: number[] }[] }[]
    }
    expect(opt.series[0].data).toHaveLength(2)
    expect(opt.series[0].data[0].name).toBe('main')
    expect(opt.series[0].data[0].value).toEqual([80, 50, 30, 60, 0, 75])
    expect(opt.series[0].data[1].name).toBe('f1')
    expect(opt.series[0].data[1].value).toEqual([50, 40, 70, 30, 0, 60])
  })

  it('seriesNameResolver override le nom', () => {
    const opt = buildRadarOption(series, {
      seriesNameResolver: (s) => `Player-${s.meta?.gamertag}`,
    }) as { series: { data: { name: string }[] }[] }
    expect(opt.series[0].data[0].name).toBe('Player-main')
  })

  it('series vide retourne option minimal', () => {
    expect(buildRadarOption([])).toEqual({ backgroundColor: 'transparent' })
  })
})

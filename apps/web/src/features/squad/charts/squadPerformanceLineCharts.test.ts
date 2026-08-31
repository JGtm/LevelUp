/**
 * squadPerformanceLineCharts.test.ts — teammates.16 (8 sous-charts).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  buildHsPerfectOption,
  buildKillsDeathsButterflyOption,
  buildPerformanceLineOption,
} from './squadPerformanceLineCharts'
import { hexComplement } from '@/lib/accessibility/hexComplement'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

vi.mock('@/lib/accessibility', async () => {
  const { hexComplement: hc } = await import('@/lib/accessibility/hexComplement')
  return {
    tokenCssVar: (token: string) => `color:${token}`,
    resolveToken: (token: string) => `hex(${token})`,
    hexComplement: hc,
  }
})

vi.mock('@/components/charts/_utils', async () => {
  const actual = await vi.importActual<typeof import('@/components/charts/_utils')>(
    '@/components/charts/_utils',
  )
  return { ...actual, seriesColor: (i: number) => `series-${i}` }
})

function pt(order: number, overrides: Partial<SquadPerformanceSeriesPoint> = {}): SquadPerformanceSeriesPoint {
  return {
    match_id: `m${order}`,
    start_time: '2026-04-30T12:00:00Z',
    match_order: order,
    kills: 10,
    deaths: 5,
    assists: 3,
    kda: 2,
    accuracy: 0.5,
    avg_life_seconds: 30,
    performance_score: 60,
    max_killing_spree: 4,
    headshot_kills: 2,
    perfect_kills: 1,
    ...overrides,
  }
}

const COLORS = { Me: '#aaa', F1: '#bbb' }
const ORDER = ['Me', 'F1']

beforeEach(() => { vi.clearAllMocks() })

describe('buildPerformanceLineOption', () => {
  it('vide → option minimale', () => {
    const opt = buildPerformanceLineOption({}, { colorByPlayer: COLORS, metric: 'assists' })
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('1 line par joueur sur la métrique demandée', () => {
    const rows = {
      Me: [pt(0, { assists: 3 }), pt(1, { assists: 5 })],
      F1: [pt(0, { assists: 1 }), pt(1, { assists: 2 })],
    }
    const opt = buildPerformanceLineOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ORDER,
      metric: 'assists',
    })
    const series = opt.series as Array<{ name: string; type: string; data: Array<number | null> }>
    expect(series).toHaveLength(2)
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1'])
    expect(series.every((s) => s.type === 'line')).toBe(true)
    expect(series[0].data).toEqual([3, 5])
    expect(series[1].data).toEqual([1, 2])
  })

  it('valeurs alignées sur match_order avec null pour les trous', () => {
    const rows = {
      Me: [pt(0, { kda: 1.5 }), pt(2, { kda: 3.0 })], // skip order=1
      F1: [pt(0, { kda: 2 }), pt(1, { kda: 2.5 }), pt(2, { kda: 1 })],
    }
    const opt = buildPerformanceLineOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ORDER,
      metric: 'kda',
      decimals: 2,
    })
    const series = opt.series as Array<{ data: Array<number | null> }>
    expect(series[0].data).toEqual([1.5, null, 3])
    expect(series[1].data).toEqual([2, 2.5, 1])
  })

  it('scale + decimals appliqués (accuracy 0..1 → %)', () => {
    const rows = { Me: [pt(0, { accuracy: 0.4567 })] }
    const opt = buildPerformanceLineOption(rows, {
      colorByPlayer: COLORS,
      metric: 'accuracy',
      decimals: 1,
      unitSuffix: ' %',
      scale: 100,
    })
    const series = opt.series as Array<{ data: Array<number | null> }>
    expect(series[0].data[0]).toBe(45.7)
  })

  it('couleur du joueur appliquée (cohérence pill/combobox)', () => {
    const rows = { Me: [pt(0)], F1: [pt(0)] }
    const opt = buildPerformanceLineOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ORDER,
      metric: 'assists',
    })
    const series = opt.series as Array<{ lineStyle: { color: string } }>
    expect(series[0].lineStyle.color).toBe('#aaa')
    expect(series[1].lineStyle.color).toBe('#bbb')
  })
})

describe('buildKillsDeathsButterflyOption', () => {
  it('2 séries par joueur (kills + deaths) ; deaths négatif', () => {
    const rows = {
      Me: [pt(0, { kills: 12, deaths: 4 }), pt(1, { kills: 8, deaths: 6 })],
    }
    const opt = buildKillsDeathsButterflyOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ['Me'],
      killsLabel: 'Frags',
      deathsLabel: 'Morts',
      hiddenTypes: new Set(['Bonus']), // défaut UI (SquadPerformanceCharts) → structure 2-séries
    })
    const series = opt.series as Array<{ name: string; type: string; data: Array<number | null> }>
    expect(series).toHaveLength(2)
    expect(series[0].name).toBe('Me — Frags')
    expect(series[0].data).toEqual([12, 8])
    expect(series[1].name).toBe('Me — Morts')
    expect(series[1].data).toEqual([-4, -6]) // négatif
  })

  it('couleur deaths = couleur complémentaire opaque (hue +180°)', () => {
    const rows = { Me: [pt(0)] }
    const opt = buildKillsDeathsButterflyOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ['Me'],
      killsLabel: 'Frags',
      deathsLabel: 'Morts',
      hiddenTypes: new Set(['Bonus']), // défaut UI → series[1] = Morts (Bonus masqué)
    })
    const series = opt.series as Array<{ itemStyle: { color: string; opacity?: number } }>
    expect(series[0].itemStyle).toMatchObject({ color: '#aaa' })
    expect(series[0].itemStyle.opacity).toBeUndefined()
    expect(series[1].itemStyle.color).toBe(hexComplement('#aaa'))
    expect(series[1].itemStyle.opacity).toBeUndefined()
  })

  describe('échelle Y stable, indépendante des toggles (item 5, DEC-5)', () => {
    type AxisOpt = { min: number; max: number }

    it('yAxis.min/max identiques AVEC et SANS la série Bonus visible (extent sur le jeu complet)', () => {
      const rows = { Me: [pt(0, { kills: 12, deaths: 4, assists: 9 })] } // bonus = 9/3 = 3
      const withoutBonus = buildKillsDeathsButterflyOption(rows, {
        colorByPlayer: COLORS,
        playerOrder: ['Me'],
        killsLabel: 'Frags',
        deathsLabel: 'Morts',
        hiddenTypes: new Set(['Bonus']),
      })
      const withBonus = buildKillsDeathsButterflyOption(rows, {
        colorByPlayer: COLORS,
        playerOrder: ['Me'],
        killsLabel: 'Frags',
        deathsLabel: 'Morts',
        hiddenTypes: new Set(), // Bonus visible
      })
      const yWithout = withoutBonus.yAxis as unknown as AxisOpt
      const yWith = withBonus.yAxis as unknown as AxisOpt
      expect(yWith.min).toBe(yWithout.min)
      expect(yWith.max).toBe(yWithout.max)
      // Valeurs exactes : max = kills(12) + bonus(3) = 15 → dizaine sup. 20.
      // min = -deaths(4) = -4 → dizaine inf. -10. Le bonus compte MÊME masqué.
      expect(yWith.max).toBe(20)
      expect(yWith.min).toBe(-10)
    })

    it('masquer un joueur ne change pas non plus l\'échelle (même cause : extent sur TOUS les joueurs)', () => {
      const rows = {
        Me: [pt(0, { kills: 12, deaths: 4, assists: 0 })],
        F1: [pt(0, { kills: 30, deaths: 2, assists: 0 })],
      }
      const shown = buildKillsDeathsButterflyOption(rows, {
        colorByPlayer: COLORS,
        playerOrder: ['Me', 'F1'],
        killsLabel: 'Frags',
        deathsLabel: 'Morts',
        hiddenTypes: new Set(['Bonus']),
      })
      const f1Hidden = buildKillsDeathsButterflyOption(rows, {
        colorByPlayer: COLORS,
        playerOrder: ['Me', 'F1'],
        killsLabel: 'Frags',
        deathsLabel: 'Morts',
        hiddenTypes: new Set(['Bonus']),
        hiddenPlayers: new Set(['F1']),
      })
      const yShown = shown.yAxis as unknown as AxisOpt
      const yHidden = f1Hidden.yAxis as unknown as AxisOpt
      expect(yHidden.min).toBe(yShown.min)
      expect(yHidden.max).toBe(yShown.max)
      // max = plus grand total positif (F1 : 30 + 0) → dizaine sup. 30.
      // min = plus grand total négatif (Me : -4) → dizaine inf. -10.
      expect(yShown.max).toBe(30)
      expect(yShown.min).toBe(-10)
    })
  })
})

describe('buildHsPerfectOption', () => {
  it('2 séries bar par joueur (HS normale + Perfect stackée avec bordure emphase)', () => {
    const rows = {
      Me: [pt(0, { headshot_kills: 4, perfect_kills: 1 })],
    }
    const opt = buildHsPerfectOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ['Me'],
      hsLabel: 'HS',
      perfectLabel: 'Perfect',
    })
    const series = opt.series as Array<{
      name: string
      type: string
      itemStyle: { color: string; borderColor?: string; borderWidth?: number; shadowBlur?: number }
      data: Array<number | null>
    }>
    expect(series).toHaveLength(2)
    expect(series[0].name).toBe('Me — HS')
    expect(series[0].type).toBe('bar')
    expect(series[0].itemStyle.color).toBe('#aaa')
    expect(series[0].itemStyle.borderColor).toBeUndefined()

    expect(series[1].name).toBe('Me — Perfect')
    expect(series[1].type).toBe('bar')
    expect(series[1].itemStyle.color).toBe('#aaa')
    expect(series[1].itemStyle.borderColor).toBeTruthy() // bordure emphase thématisée
    expect(series[1].itemStyle.borderWidth).toBe(1.5)
    expect(series[1].itemStyle.shadowBlur).toBe(8)
  })

  it('null pour valeurs manquantes (pas de fallback à 0)', () => {
    const rows = {
      Me: [pt(0, { headshot_kills: 3, perfect_kills: undefined })],
    }
    const opt = buildHsPerfectOption(rows, {
      colorByPlayer: COLORS,
      playerOrder: ['Me'],
      hsLabel: 'HS',
      perfectLabel: 'Perfect',
    })
    const series = opt.series as Array<{ data: Array<number | null> }>
    expect(series[0].data).toEqual([3]) // HS
    expect(series[1].data).toEqual([null]) // Perfect undefined → null
  })
})

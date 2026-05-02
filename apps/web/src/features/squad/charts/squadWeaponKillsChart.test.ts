/**
 * squadWeaponKillsChart.test.ts — teammates.09.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildSquadWeaponKillsOption } from './squadWeaponKillsChart'
import type { SquadWeaponKills } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
}))

const COLORS = { Me: '#aaa', F1: '#bbb' }

function data(): SquadWeaponKills {
  return {
    players: ['Me', 'F1'],
    bars: [
      { weapon_id: 1, label: 'Sniper', kills_by_player: { Me: 2, F1: 0 }, total_squad: 2 },
      { weapon_id: 2, label: 'BR75',   kills_by_player: { Me: 30, F1: 25 }, total_squad: 55 },
      { weapon_id: 3, label: 'AR',     kills_by_player: { Me: 80, F1: 60 }, total_squad: 140 },
    ],
  }
}

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadWeaponKillsOption', () => {
  it('vide → option minimale', () => {
    expect(buildSquadWeaponKillsOption(null, { colorByPlayer: COLORS })).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('aucune barre → option minimale', () => {
    expect(buildSquadWeaponKillsOption({ players: ['Me'], bars: [] }, { colorByPlayer: COLORS })).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('1 série bar par joueur', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
    const series = opt.series as Array<{ name: string; type: string; data: number[] }>
    expect(series).toHaveLength(2)
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1'])
    expect(series.every((s) => s.type === 'bar')).toBe(true)
  })

  it('valeurs alignées sur l\'ordre des bars (input garanti ASC backend)', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
    const series = opt.series as Array<{ name: string; data: number[] }>
    const me = series.find((s) => s.name === 'Me')
    expect(me?.data).toEqual([2, 30, 80])
    const f1 = series.find((s) => s.name === 'F1')
    expect(f1?.data).toEqual([0, 25, 60])
  })

  it('joueur sans entrée pour une arme → 0', () => {
    const d: SquadWeaponKills = {
      players: ['Me', 'F1'],
      bars: [{ weapon_id: 1, label: 'Sniper', kills_by_player: { Me: 5 }, total_squad: 5 }],
    }
    const opt = buildSquadWeaponKillsOption(d, { colorByPlayer: COLORS })
    const series = opt.series as Array<{ name: string; data: number[] }>
    expect(series.find((s) => s.name === 'F1')?.data).toEqual([0])
  })

  it('couleur appliquée par joueur (cohérence pill/combobox)', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
    const series = opt.series as Array<{ name: string; itemStyle: { color: string } }>
    expect(series.find((s) => s.name === 'Me')?.itemStyle.color).toBe('#aaa')
    expect(series.find((s) => s.name === 'F1')?.itemStyle.color).toBe('#bbb')
  })

  it('joueur sans couleur mappée → fallback gris #888', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: { Me: '#aaa' } })
    const series = opt.series as Array<{ name: string; itemStyle: { color: string } }>
    expect(series.find((s) => s.name === 'F1')?.itemStyle.color).toBe('#888')
  })

  it('xAxis caché (les valeurs sont dans le label de barre)', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
    const xAxis = opt.xAxis as { show: boolean }
    expect(xAxis.show).toBe(false)
  })

  it('yAxis = labels d\'armes + zebra splitArea', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
    const yAxis = opt.yAxis as { data: string[]; splitArea: { show: boolean } }
    expect(yAxis.data).toEqual(['Sniper', 'BR75', 'AR'])
    expect(yAxis.splitArea.show).toBe(true)
  })

  it('legend désactivée (pill+combobox = légende globale page)', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
    const legend = opt.legend as { show: boolean }
    expect(legend.show).toBe(false)
  })

  it('label formatter masque les zéros', () => {
    const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
    const series = opt.series as Array<{ label: { formatter: (p: { value: unknown }) => string } }>
    expect(series[0].label.formatter({ value: 0 })).toBe('')
    expect(series[0].label.formatter({ value: 12 })).toBe('12')
  })
})

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
    const series = opt.series as Array<{ label: { formatter: (p: { value: unknown; dataIndex: number }) => string } }>
    expect(series[0].label.formatter({ value: 0, dataIndex: 0 })).toBe('')
  })

  describe('I6 (V7.1) — labels % du total du joueur sur les segments assez larges', () => {
    // Me : Sniper 2, BR75 30, AR 80 → total 112 (parts ≈ 1.8 % / 26.8 % / 71.4 %).
    // F1 : Sniper 0, BR75 25, AR 60 → total 85 (parts = 0 % / 29.4 % / 70.6 %).
    it('longueurs de barres INCHANGÉES (valeurs brutes) — seul le label affiché change', () => {
      const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
      const series = opt.series as Array<{ name: string; data: number[] }>
      expect(series.find((s) => s.name === 'Me')?.data).toEqual([2, 30, 80])
      expect(series.find((s) => s.name === 'F1')?.data).toEqual([0, 25, 60])
    })

    it('segment assez large → label = % du total du joueur (arrondi), pas la valeur brute', () => {
      const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
      const series = opt.series as Array<{ name: string; label: { formatter: (p: { value: unknown; dataIndex: number }) => string } }>
      const me = series.find((s) => s.name === 'Me')!.label.formatter
      // BR75 (index 1) : 30 / 112 ≈ 26.8 % → arrondi 27 %.
      expect(me({ value: 30, dataIndex: 1 })).toBe('27 %')
      // AR (index 2) : 80 / 112 ≈ 71.4 % → arrondi 71 %.
      expect(me({ value: 80, dataIndex: 2 })).toBe('71 %')
    })

    it('segment trop étroit (< seuil de part) → label masqué, lisible seulement au tooltip', () => {
      const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
      const series = opt.series as Array<{ name: string; label: { formatter: (p: { value: unknown; dataIndex: number }) => string } }>
      const me = series.find((s) => s.name === 'Me')!.label.formatter
      // Sniper (index 0) : 2 / 112 ≈ 1.8 % — sous le seuil → masqué malgré une valeur > 0.
      expect(me({ value: 2, dataIndex: 0 })).toBe('')
    })

    it('valeur nulle → label masqué (comportement historique conservé)', () => {
      const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
      const series = opt.series as Array<{ name: string; label: { formatter: (p: { value: unknown; dataIndex: number }) => string } }>
      const f1 = series.find((s) => s.name === 'F1')!.label.formatter
      expect(f1({ value: 0, dataIndex: 0 })).toBe('')
    })

    it('tooltip enrichi : valeur brute + % du total du joueur pour chaque ligne', () => {
      const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
      const tooltip = opt.tooltip as { formatter: (raw: unknown) => string }
      const html = tooltip.formatter([
        { seriesName: 'Me', value: 30, marker: '', dataIndex: 1 },
        { seriesName: 'F1', value: 25, marker: '', dataIndex: 1 },
      ])
      expect(html).toContain('BR75')
      expect(html).toContain('Me')
      expect(html).toContain('<b>30</b>')
      expect(html).toContain('27 %') // 30/112
      expect(html).toContain('<b>25</b>')
      expect(html).toContain('29 %') // 25/85
    })

    it('tooltip : lignes à 0 exclues (même filtre que la spec existante)', () => {
      const opt = buildSquadWeaponKillsOption(data(), { colorByPlayer: COLORS })
      const tooltip = opt.tooltip as { formatter: (raw: unknown) => string }
      const html = tooltip.formatter([
        { seriesName: 'Me', value: 2, marker: '', dataIndex: 0 },
        { seriesName: 'F1', value: 0, marker: '', dataIndex: 0 },
      ])
      expect(html).toContain('Me')
      expect(html).not.toContain('F1')
    })
  })
})

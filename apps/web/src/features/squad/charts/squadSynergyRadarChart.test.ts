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

// Le contrat déclare `axes` nullable (l'API peut renvoyer un profil sans axes),
// mais cette fixture en produit TOUJOURS six. On resserre donc le type de retour
// ici plutôt que de parsemer chaque test d'assertions non-nulles : la garantie
// vient de la fixture elle-même, pas d'une supposition sur la donnée réelle.
type TestSynergyProfile = SquadSynergyRadarSeries & {
  axes: NonNullable<SquadSynergyRadarSeries['axes']>
}

function profile(player: string, values: Partial<Record<string, number>> = {}): TestSynergyProfile {
  const axes = ['combat', 'survival', 'support', 'score', 'objective', 'impact'].map((k) => ({
    axis: k,
    value: values[k] ?? 50,
    raw: 0,
  }))
  return { player, axes }
}

const COLORS = { Me: '#aaa', F1: '#bbb' }
const OPTS = { colorByPlayer: COLORS, axisLabels: AXIS_LABELS, rawLabel: 'brut' }

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadSynergyRadarOption', () => {
  it('vide → option minimale', () => {
    expect(buildSquadSynergyRadarOption([], OPTS)).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('1 trace par joueur', () => {
    const opt = buildSquadSynergyRadarOption(
      [profile('Me'), profile('F1')],
      OPTS,
    )
    const series = opt.series as Array<{ data: Array<{ name: string }> }>
    expect(series).toHaveLength(1)
    expect(series[0].data).toHaveLength(2)
    expect(series[0].data.map((d) => d.name)).toEqual(['Me', 'F1'])
  })

  it('couleur appliquée par joueur (cohérence pill/combobox)', () => {
    const opt = buildSquadSynergyRadarOption(
      [profile('Me'), profile('F1')],
      OPTS,
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
      OPTS,
    )
    const series = opt.series as Array<{ data: Array<Record<string, unknown>> }>
    expect(series[0].data[0].areaStyle).toBeUndefined()
  })

  it('axisLabels appliqués sur les indicators du radar', () => {
    const opt = buildSquadSynergyRadarOption(
      [profile('Me')],
      OPTS,
    )
    const radar = opt.radar as { indicator: Array<{ name: string; max: number }> }
    expect(radar.indicator.map((i) => i.name)).toEqual(['Combat', 'Survie', 'Support', 'Score', 'Objectif', 'Impact'])
    expect(radar.indicator.every((i) => i.max === 100)).toBe(true)
  })

  it('tooltip : valeur normalisée ET valeur brute (B3(3))', () => {
    const p = profile('Me', { combat: 80 })
    p.axes[0].raw = 1240.4 // Combat brut
    p.axes[1].raw = 1.59 // Survie brut (axe ratio → 2 décimales)
    const opt = buildSquadSynergyRadarOption([p], OPTS)
    const tooltip = opt.tooltip as {
      formatter: (p: { name: string; value: number[] }) => string
    }
    const html = tooltip.formatter({ name: 'Me', value: p.axes.map((a) => a.value) })
    expect(html).toContain('<b>80</b>')
    expect(html).toContain('brut 1240')
    expect(html).toContain('brut 1.59')
  })

  it('tooltip : valeur brute absente → seulement le normalisé (pas de NaN)', () => {
    const p = profile('Me')
    // Profil partiel : moins de valeurs brutes que d'axes affichés.
    const opt = buildSquadSynergyRadarOption([p], OPTS)
    const tooltip = opt.tooltip as {
      formatter: (p: { name: string; value: number[] }) => string
    }
    const html = tooltip.formatter({ name: 'Inconnu', value: p.axes.map((a) => a.value) })
    expect(html).not.toContain('NaN')
    expect(html).not.toContain('brut')
  })

  it('joueur sans couleur mappée → fallback gris #888', () => {
    const opt = buildSquadSynergyRadarOption(
      [profile('Unknown')],
      OPTS,
    )
    const series = opt.series as Array<{ data: Array<{ itemStyle: { color: string } }> }>
    expect(series[0].data[0].itemStyle.color).toBe('#888')
  })
})

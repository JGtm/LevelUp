/**
 * FragWeaponBreakdown.test.tsx (P1.5/P1.7) — builder pur : barres par arme, une
 * couleur par classe (via fragClassColor), classe portée dans le tooltip. Rendu
 * mocké echarts-for-react (jsdom sans canvas).
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { FragWeaponBreakdown, buildFragWeaponBreakdownOption, type FragWeaponLabels } from './FragWeaponBreakdown'
import type { SynthesisWeaponKillEntry } from '@/lib/api/types'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const LABELS: FragWeaponLabels = {
  classLabel: (c) => `class:${c}`,
  formatValue: (n) => String(n),
  killsSuffix: 'frags',
}

const WEAPONS: SynthesisWeaponKillEntry[] = [
  { label: 'BR75', kills: 12, class: 'shoulder', role: 'precision' },
  { label: 'Épée', kills: 4, class: 'melee', role: 'melee' },
  { label: 'Inconnue', kills: 2 },
]

describe('buildFragWeaponBreakdownOption', () => {
  it('barres horizontales, plus grande en haut, classe portée par datum', () => {
    const opt = buildFragWeaponBreakdownOption(WEAPONS, LABELS) as {
      series: { type: string; data: Array<{ value: number; className?: string; itemStyle: Record<string, unknown> }> }[]
      yAxis: { data: string[] }
    }
    const s = opt.series[0]
    expect(s.type).toBe('bar')
    expect(s.data).toHaveLength(3)
    // reverse() → la plus grande (BR75) en dernier index (haut de l'axe catégoriel).
    expect(opt.yAxis.data[opt.yAxis.data.length - 1]).toBe('BR75')
    const br = s.data.find((d) => d.className === 'class:shoulder')
    expect(br).toBeDefined()
    // Arme sans classe → pas de className (fallback couleur neutre géré par fragClassColor).
    expect(s.data.some((d) => d.className === undefined)).toBe(true)
  })

  it('liste vide → aucune série', () => {
    const opt = buildFragWeaponBreakdownOption([], LABELS) as { series?: unknown[] }
    expect(opt.series).toBeUndefined()
  })

  it('I4 : légende des classes en bas, centrée (une entrée par classe représentée, ordre de première apparition)', () => {
    const opt = buildFragWeaponBreakdownOption(WEAPONS, LABELS) as {
      legend: { left?: string; bottom?: number; data?: string[] }
      grid: { bottom: number }
    }
    // reverse() → ordre de balayage Inconnue (pas de classe), Épée (melee), BR75 (shoulder).
    expect(opt.legend.data).toEqual(['class:melee', 'class:shoulder'])
    expect(opt.legend.left).toBe('center')
    expect(opt.legend.bottom).toBe(0)
    // Le grid réserve de la place en bas pour la légende (pas de chevauchement).
    expect(opt.grid.bottom).toBeGreaterThan(8)
  })

  it('I4 : la série réelle reste en premier (contrat testé), les séries fantômes de légende suivent', () => {
    const opt = buildFragWeaponBreakdownOption(WEAPONS, LABELS) as {
      series: Array<{ name?: string; data: unknown[]; silent?: boolean }>
    }
    expect(opt.series[0].data).toHaveLength(3) // série réelle (données des barres)
    const ghosts = opt.series.slice(1)
    expect(ghosts).toHaveLength(2)
    for (const g of ghosts) {
      expect(g.data).toHaveLength(0)
      expect(g.silent).toBe(true)
    }
  })

  it('I4 : aucune arme avec classe résolue → pas de légende, grid inchangé', () => {
    const noClass: SynthesisWeaponKillEntry[] = [{ label: 'Inconnue', kills: 2 }]
    const opt = buildFragWeaponBreakdownOption(noClass, LABELS) as {
      legend: { show?: boolean }
      grid: { bottom: number }
    }
    expect(opt.legend.show).toBe(false)
    expect(opt.grid.bottom).toBe(8)
  })

  it('survol lié : hoveredClass estompe les armes des AUTRES classes, garde la classe survolée', () => {
    const opt = buildFragWeaponBreakdownOption(WEAPONS, LABELS, 'shoulder') as {
      series: { data: Array<{ classKey?: string; itemStyle: { opacity: number } }> }[]
    }
    const data = opt.series[0].data
    const br = data.find((d) => d.classKey === 'shoulder')!
    const sword = data.find((d) => d.classKey === 'melee')!
    const unknown = data.find((d) => d.classKey === undefined)!
    expect(br.itemStyle.opacity).toBe(1) // classe survolée → pleine opacité
    expect(sword.itemStyle.opacity).toBeCloseTo(0.28) // autre classe → estompée
    expect(unknown.itemStyle.opacity).toBeCloseTo(0.28) // arme sans classe → estompée
  })

  it('sans hoveredClass : aucune arme estompée (autonome)', () => {
    const opt = buildFragWeaponBreakdownOption(WEAPONS, LABELS) as {
      series: { data: Array<{ itemStyle: { opacity: number } }> }[]
    }
    for (const d of opt.series[0].data) expect(d.itemStyle.opacity).toBe(1)
  })
})

describe('FragWeaponBreakdown (composant)', () => {
  it('avec armes → monte le chart', async () => {
    render(<FragWeaponBreakdown weapons={WEAPONS} />)
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
  })
})

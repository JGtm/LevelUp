/**
 * FragSunburst.test.tsx (P1.7) — rendu du composant (echarts-for-react mocké :
 * jsdom n'a pas de canvas, cf. convention repo) + contrat du builder pur
 * (hiérarchie classe→rôle, résidu hachuré, null si total 0).
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { FragSunburst, buildFragSunburstOption, type FragSunburstLabels } from './FragSunburst'
import type { FragDistribution } from '@/lib/api/types'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const LABELS: FragSunburstLabels = {
  classLabel: (c) => `class:${c}`,
  roleLabel: (r) => `role:${r}`,
  centerLabel: 'Frags',
  authorityExact: 'exact',
  authorityEstimated: 'estimé',
  formatValue: (n) => String(n),
  shareTotal: (pct) => `${pct}% total`,
  shareClass: (pct, cn) => `${pct}% ${cn}`,
}

const DIST: FragDistribution = {
  total_kills: 18,
  classes: [
    { class: 'shoulder', kills: 10, authoritative: false, roles: [
      { role: 'precision', kills: 6 },
      { role: 'automatic', kills: 4 },
    ] },
    { class: 'melee', kills: 5, authoritative: true },
    { class: 'unattributed', kills: 3, authoritative: false },
  ],
}

describe('buildFragSunburstOption (builder pur)', () => {
  it('produit un sunburst 2 niveaux, ordre conservé, résidu hachuré', () => {
    const opt = buildFragSunburstOption(DIST.classes ?? [], DIST.total_kills, LABELS) as {
      series: { type: string; data: Array<{ name: string; value: number; children?: unknown[]; itemStyle: Record<string, unknown> }> }[]
    }
    const s = opt.series[0]
    expect(s.type).toBe('sunburst')
    expect(s.data).toHaveLength(3)
    // Classe avec rôles → children niveau 2.
    expect(s.data[0].children).toHaveLength(2)
    // Classe feuille (melee sans rôles) → pas de children.
    expect(s.data[1].children).toBeUndefined()
    // Non attribué → décal (hachure).
    expect(s.data[2].itemStyle.decal).toBeDefined()
  })

  it('total 0 → aucune série (rien à tracer)', () => {
    const opt = buildFragSunburstOption([], 0, LABELS) as { series?: unknown[] }
    expect(opt.series).toBeUndefined()
  })
})

describe('FragSunburst (composant)', () => {
  it('total > 0 → monte le chart (titre + canvas mocké)', async () => {
    render(<FragSunburst distribution={DIST} />)
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
  })

  it('total 0 → rend null (aucune carte)', () => {
    const { container } = render(<FragSunburst distribution={{ total_kills: 0, classes: [] }} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('distribution absente → rend null', () => {
    const { container } = render(<FragSunburst distribution={null} />)
    expect(container).toBeEmptyDOMElement()
  })
})

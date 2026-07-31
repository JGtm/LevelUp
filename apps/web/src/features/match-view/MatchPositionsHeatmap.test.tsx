import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MatchPositionsHeatmap, binPositionsToHeatmap } from './MatchPositionsHeatmap'
import type { MatchPlayerPosition } from '@/lib/api/types'

// Mock echarts-for-react : jsdom ne supporte pas le canvas, et un chart monté
// trop longtemps crashe (canvas non rattrapable). On stub le rendu.
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

const sample: MatchPlayerPosition[] = [
  { timeMs: 0, x: 0, y: 0, z: 0, team: 0 },
  { timeMs: 0, x: 0.5, y: 0.5, z: 0, team: 0 },
  { timeMs: 20000, x: 100, y: 100, z: 0, team: 1 },
]

describe('binPositionsToHeatmap', () => {
  it('retourne [] pour un set vide', () => {
    expect(binPositionsToHeatmap([])).toEqual([])
  })

  it('regroupe les positions proches dans la même cellule', () => {
    // (0,0) et (0.5,0.5) tombent dans le bin 0 (span 100 / 20 = 5 par cellule).
    const out = binPositionsToHeatmap(sample)
    // 2 cellules non vides : le coin bas-gauche (densité 2) + le coin haut (densité 1).
    expect(out.length).toBe(2)
    const dense = out.find((d) => d.value === 2)
    expect(dense).toBeTruthy()
    expect(dense?.detail?.count).toBe(2)
  })

  it('tolère un span dégénéré (positions colinéaires) sans NaN', () => {
    const colinear: MatchPlayerPosition[] = [
      { timeMs: 0, x: 5, y: 0, z: 0, team: -1 },
      { timeMs: 0, x: 5, y: 0, z: 0, team: -1 },
    ]
    const out = binPositionsToHeatmap(colinear)
    expect(out.length).toBe(1)
    expect(out[0].value).toBe(2)
    expect(Number.isNaN(Number(out[0].x))).toBe(false)
  })
})

describe('MatchPositionsHeatmap', () => {
  it('rend sans crash avec des positions (echarts mocké)', async () => {
    render(<MatchPositionsHeatmap positions={sample} locale="fr" />)
    expect(screen.getByText('Heatmap des positions')).toBeTruthy()
    // ChartCard lazy-load echarts-for-react via Suspense : le stub n'est monté
    // qu'après résolution du lazy import → assertion asynchrone.
    expect(await screen.findByTestId('echarts-stub')).toBeTruthy()
  })

  it('affiche le filtre par équipe quand au moins une position a team != -1', () => {
    render(<MatchPositionsHeatmap positions={sample} locale="fr" />)
    expect(screen.getByText('Équipe A')).toBeTruthy()
    expect(screen.getByText('Équipe B')).toBeTruthy()
    expect(screen.getByText('Global')).toBeTruthy()
  })

  it('masque le filtre équipe quand toutes les positions sont team=-1', () => {
    const unknown: MatchPlayerPosition[] = [
      { timeMs: 0, x: 1, y: 2, z: 0, team: -1 },
      { timeMs: 0, x: 3, y: 4, z: 0, team: -1 },
    ]
    render(<MatchPositionsHeatmap positions={unknown} locale="fr" />)
    expect(screen.queryByText('Équipe A')).toBeNull()
  })

  it('se masque proprement (null) sans positions', () => {
    const { container } = render(<MatchPositionsHeatmap positions={[]} locale="fr" />)
    expect(container.firstChild).toBeNull()
  })

  it('se masque proprement (null) quand positions est undefined', () => {
    const { container } = render(<MatchPositionsHeatmap positions={undefined} locale="en" />)
    expect(container.firstChild).toBeNull()
  })

  it('rend le titre EN', () => {
    render(<MatchPositionsHeatmap positions={sample} locale="en" />)
    expect(screen.getByText('Positions heatmap')).toBeTruthy()
  })
})

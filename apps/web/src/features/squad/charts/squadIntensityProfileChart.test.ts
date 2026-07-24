/**
 * squadIntensityProfileChart.test.ts — « Intensité » profil médian + enveloppe.
 *
 * Couvre : layout des grilles (N=1 pleine largeur, 2/3/4 → 2 colonnes), échelle
 * Y partagée entre panneaux, panneau absent si aucune manche exploitable, médiane
 * seule (pas d'enveloppe) sous MIN_MATCHES_FOR_ENVELOPE manches.
 */
import { describe, it, expect } from 'vitest'

import {
  buildSquadIntensityProfileOption,
  computeGrids,
  type IntensityPanelInput,
} from './squadIntensityProfileChart'

/** N manches concentrées sur `phaseIdx` (part = 1 sur cette phase). */
function rows(n: number, phaseIdx = 0): Array<{ phases: number[] }> {
  return Array.from({ length: n }, () => {
    const p = new Array<number>(10).fill(0)
    p[phaseIdx] = 1
    return { phases: p }
  })
}

function panel(key: string, n: number, phaseIdx = 0): IntensityPanelInput {
  return { key, label: key, color: '#ff8800', rows: rows(n, phaseIdx) }
}

const OPTS = { medianLabel: 'Médiane', envelopeLabel: 'Enveloppe P25–P75', refLabel: '10 %' }

function seriesIds(opt: ReturnType<typeof buildSquadIntensityProfileOption>): string[] {
  return (opt.series as Array<{ id?: string }>).map((s) => String(s.id ?? ''))
}

describe('computeGrids', () => {
  it('N=1 → un panneau pleine largeur (1 colonne)', () => {
    const g = computeGrids(1)
    expect(g).toHaveLength(1)
    // 1 colonne : largeur = 100 - left(6) - right(3) = 91 %.
    expect(g[0].width).toBe('91%')
  })

  it('N=2 → une rangée de 2 (même top)', () => {
    const g = computeGrids(2)
    expect(g).toHaveLength(2)
    expect(g[0].top).toBe(g[1].top)
    expect(g[0].left).not.toBe(g[1].left)
  })

  it('N=3 → 2+1 (le 3e panneau sur une 2e rangée)', () => {
    const g = computeGrids(3)
    expect(g).toHaveLength(3)
    expect(g[0].top).toBe(g[1].top)
    expect(g[2].top).not.toBe(g[0].top)
  })

  it('N=4 → 2×2 (deux rangées de 2)', () => {
    const g = computeGrids(4)
    expect(g).toHaveLength(4)
    expect(g[0].top).toBe(g[1].top)
    expect(g[2].top).toBe(g[3].top)
    expect(g[2].top).not.toBe(g[0].top)
  })
})

describe('buildSquadIntensityProfileOption', () => {
  it('aucun panneau exploitable → option minimale (fond seul)', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [{ key: 'A', label: 'A', color: '#fff', rows: [{ phases: null }] }],
      ...OPTS,
    })
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
    expect(opt.series).toBeUndefined()
  })

  it('1 joueur (≥4 manches) → 1 grille + base/bande/médiane', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    expect(opt.grid).toHaveLength(1)
    expect(opt.title).toHaveLength(1)
    expect(seriesIds(opt)).toEqual(['base-0', 'band-0', 'median-0'])
  })

  it('4 joueurs → 4 grilles, 4 axes X/Y, titres = gamertags', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5), panel('B', 5), panel('C', 5), panel('D', 5)],
      ...OPTS,
    })
    expect(opt.grid).toHaveLength(4)
    expect(opt.xAxis).toHaveLength(4)
    expect(opt.yAxis).toHaveLength(4)
    const titles = (opt.title as Array<{ text: string }>).map((t) => t.text)
    expect(titles).toEqual(['A', 'B', 'C', 'D'])
  })

  it('échelle Y PARTAGÉE : tous les panneaux ont le même max', () => {
    // A concentré phase 0 (part médiane haute), B étalé → max commun.
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5, 0), panel('B', 5, 3)],
      ...OPTS,
    })
    const maxes = (opt.yAxis as Array<{ max: number }>).map((y) => y.max)
    expect(maxes[0]).toBe(maxes[1])
    expect(maxes[0]).toBeGreaterThan(0)
  })

  it('panneau absent si le joueur n a aucune manche exploitable', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [
        panel('A', 5),
        { key: 'B', label: 'B', color: '#0af', rows: [{ phases: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0] }] },
      ],
      ...OPTS,
    })
    expect(opt.grid).toHaveLength(1)
    expect((opt.title as Array<{ text: string }>)[0].text).toBe('A')
  })

  it('médiane seule (< 4 manches) : pas de base/bande d enveloppe', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 3)], ...OPTS })
    expect(seriesIds(opt)).toEqual(['median-0'])
  })

  it('repère 10 % : markLine sur la médiane à 1/PHASE_COUNT', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 3)], ...OPTS })
    const median = (opt.series as Array<{ id?: string; markLine?: { data: Array<{ yAxis: number }> } }>).find(
      (s) => s.id === 'median-0',
    )
    expect(median?.markLine?.data[0].yAxis).toBeCloseTo(0.1)
  })
})

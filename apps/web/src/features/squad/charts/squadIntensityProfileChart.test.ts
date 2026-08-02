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

const OPTS = {
  medianLabel: 'Médiane',
  envelopeLabel: 'Enveloppe P25–P75',
  refLabel: '10 %',
  axisLabels: { start: 'Début', mid: 'Milieu', end: 'Fin', rangeSuffix: 'du match' },
}

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

  it('axe X : 3 repères seulement (Début/Milieu/Fin aux index 0/5/9)', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    const xAxis = (opt.xAxis as Array<{
      data: string[]
      axisLabel: {
        interval: (i: number) => boolean
        formatter: (v: string, i: number) => string
      }
    }>)[0]
    // Les 10 tranches restent en data (tooltip précis).
    expect(xAxis.data).toHaveLength(10)
    // interval : vrai seulement aux index 0, 5, 9.
    const shown = [...Array(10).keys()].filter((i) => xAxis.axisLabel.interval(i))
    expect(shown).toEqual([0, 5, 9])
    // formatter : libellés i18n aux 3 repères, vide ailleurs.
    expect(xAxis.axisLabel.formatter('', 0)).toBe('Début')
    expect(xAxis.axisLabel.formatter('', 5)).toBe('Milieu')
    expect(xAxis.axisLabel.formatter('', 9)).toBe('Fin')
    expect(xAxis.axisLabel.formatter('', 4)).toBe('')
  })

  it('axe Y : formatter en pourcentage (0.1 → « 10% »)', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    const yAxis = (opt.yAxis as Array<{ axisLabel: { formatter: (v: number) => string } }>)[0]
    expect(yAxis.axisLabel.formatter(0.1)).toBe('10%')
  })

  it('tooltip : tranche précise via dataIndex + suffixe (« 40-50% du match »)', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    const formatter = (opt.tooltip as { formatter: (p: unknown) => string }).formatter
    const html = formatter([{ seriesId: 'median-0', seriesName: 'A', dataIndex: 4, value: 0.3 }])
    expect(html).toContain('40-50% du match')
    expect(html).toContain('Médiane')
  })

  // Affordance de survol (v7.3 lot 2, item 2.3c) : sans symbole survolable, le
  // canvas se lit comme une image figée. Symbole caché au repos (showSymbol:false)
  // mais DÉFINI (symbol ≠ 'none'), sinon ECharts n'affiche rien à l'emphase.
  it('médiane : symbole caché au repos mais révélé au survol', () => {
    const opt = buildSquadIntensityProfileOption({ panels: [panel('A', 5)], ...OPTS })
    const median = (opt.series as Array<{ id?: string; showSymbol?: boolean; symbol?: string }>).find(
      (s) => s.id === 'median-0',
    )
    expect(median?.showSymbol).toBe(false)
    expect(median?.symbol).toBe('circle')
  })
})

describe('buildSquadIntensityProfileOption — courbe agrégée d équipe', () => {
  const TEAM = { label: 'Équipe', rows: rows(8, 5) }

  it('teamOverlay absent : rendu inchangé (aucune série team)', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5), panel('B', 5), panel('C', 5)],
      ...OPTS,
    })
    expect(seriesIds(opt).some((id) => id.startsWith('team-'))).toBe(false)
  })

  it('teamOverlay fourni : une courbe d équipe PAR panneau, liée à sa grille', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5), panel('B', 5), panel('C', 5)],
      ...OPTS,
      teamOverlay: TEAM,
    })
    const team = (opt.series as Array<{ id?: string; xAxisIndex?: number; name?: string }>).filter((s) =>
      String(s.id ?? '').startsWith('team-'),
    )
    expect(team).toHaveLength(3)
    expect(team.map((s) => s.xAxisIndex)).toEqual([0, 1, 2])
    expect(team[0].name).toBe('Équipe')
  })

  it('la médiane d équipe vient du helper canonique (manches `all`, pas une moyenne de médianes)', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5, 0), panel('B', 5, 0), panel('C', 5, 0)],
      ...OPTS,
      teamOverlay: TEAM,
    })
    const team = (opt.series as Array<{ id?: string; data?: number[] }>).find((s) => s.id === 'team-0')
    // TEAM = 8 manches concentrées sur la phase 5 → médiane 1 en phase 5, 0 ailleurs
    // (une moyenne des médianes joueur aurait donné le profil de la phase 0).
    expect(team?.data?.[5]).toBeCloseTo(1)
    expect(team?.data?.[0]).toBeCloseTo(0)
  })

  it('échelle Y partagée : la courbe d équipe entre dans le cadre', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5, 0), panel('B', 5, 0), panel('C', 5, 0)],
      ...OPTS,
      teamOverlay: TEAM,
    })
    const yMax = (opt.yAxis as Array<{ max: number }>)[0].max
    expect(yMax).toBeGreaterThanOrEqual(1)
  })

  it('manches d équipe sans frag : aucune courbe d équipe plate', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5), panel('B', 5), panel('C', 5)],
      ...OPTS,
      teamOverlay: { label: 'Équipe', rows: [{ phases: new Array<number>(10).fill(0) }] },
    })
    expect(seriesIds(opt).some((id) => id.startsWith('team-'))).toBe(false)
  })

  it('tooltip : ligne « Équipe » ajoutée sous la médiane du joueur', () => {
    const opt = buildSquadIntensityProfileOption({
      panels: [panel('A', 5), panel('B', 5), panel('C', 5)],
      ...OPTS,
      teamOverlay: TEAM,
    })
    const formatter = (opt.tooltip as { formatter: (p: unknown) => string }).formatter
    const html = formatter([
      { seriesId: 'median-0', seriesName: 'A', dataIndex: 4, value: 0.3 },
      { seriesId: 'team-0', seriesName: 'Équipe', dataIndex: 4, value: 0.2 },
    ])
    expect(html).toContain('Équipe : 20%')
  })
})

/**
 * squadRankStripsChart.test.ts — le sous-graphe « Rang » en bandeaux : une grille par joueur
 * (abscisse partagée), la couleur du joueur seule, triangle plein vers le haut / creux vers le
 * bas, étiquette « rang (écart) », bornes à deux graduations, MMR d'équipe en bandeau séparé,
 * aucune couleur opposée.
 */
import { afterAll, beforeAll, describe, expect, it } from 'vitest'

import { renderOptionToSvg } from '@/components/charts/svgRender.testutil'
import { _resetActivePalette, applyPalette } from '@/lib/accessibility/applyPalette'
import { defaultPalette } from '@/lib/accessibility/palettes/default'
import { hexComplement } from '@/lib/accessibility/hexComplement'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import {
  buildRankStripsOption,
  rankStripsCount,
  rankStripsHeight,
  STRIP_GAP,
  STRIP_HEIGHT,
  type RankStripsOpts,
} from './squadRankStripsChart'

const pt = (order: number, rating: number, delta: number | undefined, mmr?: number): SquadPerformanceSeriesPoint =>
  ({
    match_order: order,
    match_id: `m${order}`,
    start_time: '2026-09-10T20:00:00Z',
    kills: 10,
    deaths: 8,
    assists: 3,
    skill_rating: rating,
    skill_rating_type: 'csr',
    skill_delta: delta,
    team_mmr: mmr,
  }) as SquadPerformanceSeriesPoint

const ROWS = {
  JGtm: [pt(0, 1500, 12, 1400), pt(1, 1491, -9, 1410), pt(2, 1512, 21, 1420)],
  Kaya: [pt(0, 1200, 4), pt(1, 1204, 4), pt(2, 1215, 11)],
}
const PLAYERS = ['JGtm', 'Kaya']
const OPTS: RankStripsOpts = {
  colorByPlayer: { JGtm: '#112233', Kaya: '#445566' },
  xLabels: ['#1\nStreets', '#2\nBazaar', '#3'],
  mmrLabel: 'MMR équipe',
  upLabel: 'rang tenu ou gagné',
  downLabel: 'rang perdu',
  changeLabel: (d) => `variation ${d}`,
  fmtInt: (v) => new Intl.NumberFormat('fr-FR', { maximumFractionDigits: 0 }).format(v),
  fmtDelta: (v) => (v > 0 ? `+${v}` : String(v)),
}

type Datum = {
  value: number
  symbol: string
  symbolRotate: number
  itemStyle: { color: string; borderColor: string }
  tip: string
} | null
type Series = {
  type: string
  name: string
  xAxisIndex?: number
  yAxisIndex?: number
  data: Datum[]
  lineStyle?: { color: string; type?: string }
}
interface Opt {
  title: { text: string; subtext: string; textStyle: { color: string } }[]
  grid: { top: number; height: number }[]
  xAxis: { gridIndex: number; axisLabel: { show: boolean } }[]
  yAxis: { gridIndex: number; min: number; max: number; interval: number; axisLabel: { formatter: (v: number) => string } }[]
  series: Series[]
  legend: { data: { name: string; icon: string; itemStyle: { color: string } }[] }
  tooltip: { formatter: (p: unknown) => string }
}
const build = () => buildRankStripsOption(ROWS, PLAYERS, 3, OPTS) as unknown as Opt

describe('buildRankStripsOption', () => {
  beforeAll(() => applyPalette(defaultPalette, 'default'))
  afterAll(() => _resetActivePalette())

  it('sans rang ni MMR : option vide', () => {
    const none = { A: [{ ...pt(0, 0, 0), skill_rating: undefined, team_mmr: undefined } as SquadPerformanceSeriesPoint] }
    expect(buildRankStripsOption(none, ['A'], 1, OPTS)).toEqual({ backgroundColor: 'transparent' })
  })

  it('une grille par bandeau : un par joueur plus le MMR d’équipe, empilées, abscisse partagée', () => {
    const o = build()
    expect(o.grid).toHaveLength(3)
    expect(o.grid.map((g) => g.height)).toEqual([STRIP_HEIGHT, STRIP_HEIGHT, STRIP_HEIGHT])
    expect(o.grid[1].top - o.grid[0].top).toBe(STRIP_HEIGHT + STRIP_GAP)
    expect(o.xAxis.map((a) => a.axisLabel.show)).toEqual([false, false, true])
    expect(rankStripsCount(ROWS, PLAYERS, 3)).toBe(3)
    expect(rankStripsHeight(3)).toBeGreaterThan(rankStripsHeight(2))
  })

  it('étiquette de gauche : nom à la couleur du joueur, rang de fin et écart sur la session', () => {
    const o = build()
    expect(o.title.map((t) => t.text)).toEqual(['JGtm', 'Kaya', 'MMR équipe'])
    expect(o.title[0].textStyle.color).toBe('#112233')
    // JGtm : 1 512 ; écart 1 512 − (1 500 − 12) = 24. Kaya : 1 215 ; 1 215 − (1 200 − 4) = 19.
    expect(o.title[0].subtext).toBe('1 512 (+24)')
    expect(o.title[1].subtext).toBe('1 215 (+19)')
    expect(o.title[2].subtext).toBe('1 420')
  })

  it('même amplitude pour tous les bandeaux de joueur, deux graduations seulement (les bornes)', () => {
    const o = build()
    const [a, b] = o.yAxis
    expect(a.max - a.min).toBe(b.max - b.min)
    expect(a.interval).toBe(a.max - a.min)
    // Étendues 21 et 11 : demi-amplitude 10,5 + max(3,15 ; 4) = 14,5 -> 15.
    expect(a.max - a.min).toBe(30)
    expect(a.min).toBe(1502 - 15) // centre de JGtm : (1491 + 1512) / 2 = 1501,5 -> 1502
    expect(b.min).toBe(1208 - 15) // centre de Kaya : (1200 + 1215) / 2 = 1207,5 -> 1208
    expect(a.axisLabel.formatter(1487)).toBe('1 487')
  })

  it('le MMR d’équipe a sa propre échelle, dans le dernier bandeau', () => {
    const mmr = build().yAxis[2]
    expect(mmr.min).toBeLessThanOrEqual(1400)
    expect(mmr.max).toBeGreaterThanOrEqual(1420)
    expect(mmr.max - mmr.min).not.toBe(build().yAxis[0].max - build().yAxis[0].min)
  })

  it('courbe du joueur : sa seule couleur ; MMR : tireté, couleur du thème, sans la couleur d’un joueur', () => {
    const o = build()
    const jg = o.series.find((s) => s.name === 'JGtm — CSR')!
    expect(jg.lineStyle!.color).toBe('#112233')
    expect(jg.xAxisIndex).toBe(0)
    const mmr = o.series.find((s) => s.name === 'MMR équipe')!
    expect(mmr.lineStyle!.type).toBe('dashed')
    expect(mmr.lineStyle!.color).not.toBe('#112233')
    expect(mmr.xAxisIndex).toBe(2)
  })

  it('marqueurs : triangle plein vers le haut si le rang tient ou monte, creux vers le bas s’il recule', () => {
    const jg = build().series.find((s) => s.name === 'JGtm — CSR')!
    const [up, down, up2] = jg.data as NonNullable<Datum>[]
    expect(up).toMatchObject({ symbol: 'triangle', symbolRotate: 0, itemStyle: { color: '#112233', borderColor: '#112233' } })
    expect(down).toMatchObject({ symbol: 'triangle', symbolRotate: 180, itemStyle: { color: 'transparent', borderColor: '#112233' } })
    expect(up2.symbolRotate).toBe(0)
  })

  it('un delta à zéro est un triangle plein ; sans variation servie, un rond plein', () => {
    const rows = { JGtm: [pt(0, 1500, 0), pt(1, 1500, undefined)] }
    const o = buildRankStripsOption(rows, ['JGtm'], 2, OPTS) as unknown as Opt
    const a = o.series.find((s) => s.name === 'JGtm — CSR')!.data as NonNullable<Datum>[]
    expect(a[0]).toMatchObject({ symbol: 'triangle', symbolRotate: 0, itemStyle: { color: '#112233' } })
    expect(a[1].symbol).toBe('circle')
  })

  it('aucune couleur opposée : le rendu ne pose que la couleur de chaque joueur, jamais son complément', () => {
    const svg = renderOptionToSvg(buildRankStripsOption(ROWS, PLAYERS, 3, OPTS), 900, rankStripsHeight(3)).toLowerCase()
    expect(svg).toContain('#112233')
    expect(svg).toContain('#445566')
    // L’opposé colorimétrique d’une couleur de joueur n’apparaît pas (il ressemblait à la teinte du joueur suivant).
    expect(svg).not.toContain(hexComplement('#112233').toLowerCase())
    expect(svg).not.toContain(hexComplement('#445566').toLowerCase())
  })

  it('infobulle d’un marqueur : joueur, match, type et rang, variation signée', () => {
    const o = build()
    const jg = o.series.find((s) => s.name === 'JGtm — CSR')!
    expect(o.tooltip.formatter({ data: jg.data[1] })).toBe(
      '<b>JGtm</b><br/>#2 · Bazaar<br/>CSR 1 491<br/>variation -9',
    )
    const mmr = o.series.find((s) => s.name === 'MMR équipe')!
    expect(o.tooltip.formatter({ data: mmr.data[0] })).toBe('<b>MMR équipe</b><br/>#1 · Streets<br/>1 400')
    expect(o.tooltip.formatter({ data: undefined })).toBe('')
  })

  it('légende : seulement les deux sens de marqueur, plein vers le haut et creux vers le bas', () => {
    const l = build().legend
    expect(l.data.map((d) => d.name)).toEqual(['rang tenu ou gagné', 'rang perdu'])
    expect(l.data[0].icon).toBe('triangle')
    expect(l.data[1].icon.startsWith('path://')).toBe(true)
    expect(l.data[1].itemStyle.color).toBe('transparent')
  })

  it('un joueur sans rang n’a pas de bandeau ; sans MMR, pas de bandeau MMR', () => {
    const rows = { A: [pt(0, 1500, 1)], B: [{ ...pt(0, 0, 0), skill_rating: undefined } as SquadPerformanceSeriesPoint] }
    const o = buildRankStripsOption(rows, ['A', 'B'], 1, OPTS) as unknown as Opt
    expect(o.title.map((t) => t.text)).toEqual(['A'])
    expect(o.grid).toHaveLength(1)
  })
})

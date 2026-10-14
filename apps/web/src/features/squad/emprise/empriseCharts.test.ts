/**
 * empriseCharts.test.ts — « Contrôle des ressources au fil de la session » (22/09) : une courbe
 * cumulée par ressource du bilan, valeur au bout, points par match à la taille de leur volume,
 * trait 50 %, heure et carte sous chaque match, bande de résultats et encoche de dominance.
 */
import { describe, expect, it } from 'vitest'

import { axisFootProblems } from '@/components/charts/axisFoot.check'

import { buildResourceFilOption, pickupRadius, type EmpriseFilColors, type EmpriseFilText } from './empriseCharts'
import { buildResourceFil, empriseMatchIndex } from './emprise.logic'
import { EMPRISE_2209, HISTORY_2209 } from './emprise.fixtures'
import { EMPRISE_TEXT } from './empriseStrings'
import { inSentence } from './useOutcomeLabels'

const T = EMPRISE_TEXT.fr
/** Libellés d'issue du manifest du titre (`outcomes.toml`, fr). */
const OUTCOME_FR = { win: 'Victoire', loss: 'Défaite', tie: 'Égalité', dnf: 'Abandon' } as const
const COLORS: EmpriseFilColors = {
  resource: (r) => `res-${r}`,
  win: 'win',
  loss: 'loss',
  neutral: 'neutral',
  parity: 'parity',
  dominance: (d) => `dom${d}`,
  theme: {
    axisLabel: 'axis',
    axisLine: 'line',
    splitLine: 'split',
    splitAreaA: 'a',
    splitAreaB: 'b',
    text: 'text',
    tooltipBg: 'bg',
    tooltipBorder: 'border',
    card: 'card',
    isDark: false,
  },
}

type Item = { value: unknown; symbolSize?: number; tip?: string; itemStyle?: { color?: string } }
type Series = {
  type: string
  name?: string
  data: (Item | number | null)[]
  lineStyle?: { color: string }
  markLine?: { data: { yAxis: number }[] }
  endLabel?: { formatter: (p: { value: unknown }) => string }
  renderItem?: (p: unknown, api: unknown) => { children: { style: { fill: string } }[] }
}
type Axis = { axisLabel?: { formatter: (v: string, i: number) => string } }

const fil = buildResourceFil(EMPRISE_2209, empriseMatchIndex(HISTORY_2209))
const T_FIL: EmpriseFilText = {
  resourceLabel: (r) => T.resources[r].label,
  pctFmt: T.pctFmt,
  pctIntFmt: T.pctIntFmt,
  timeOf: (iso) => iso.slice(11, 16),
  outcomeOf: (m) => (m.outcome ? inSentence(OUTCOME_FR[m.outcome]) : null),
  resultOf: (m) => (m.outcome ? `${OUTCOME_FR[m.outcome]} ${m.score}` : null),
  dominanceLabel: (d) => (d === 1 ? 'Domination' : String(d)),
  pointTip: T.fil.pointTip,
  endTip: T.fil.endTip,
  bandTip: T.fil.bandTip,
}
const opt = buildResourceFilOption(fil, COLORS, T_FIL) as { series: Series[]; xAxis: Axis[] }

const lines = opt.series.filter((s) => s.type === 'line')
const dots = opt.series.filter((s) => s.type === 'scatter')

describe('buildResourceFilOption — 22/09', () => {
  it('le pied en mode match (heure + carte, bande) tient dans le bas de la grille', () => {
    expect(axisFootProblems(opt as never)).toEqual([])
  })

  it('une courbe par ressource du bilan, à sa couleur, dans l’ordre (bonus, armes spéciales)', () => {
    expect(lines.map((s) => s.name)).toEqual(['Bonus', 'Armes spéciales'])
    expect(lines.map((s) => s.lineStyle?.color)).toEqual(['res-powerup', 'res-power_weapon'])
  })

  it('la courbe est le cumul ; le match sans la ressource n’a pas de point (la courbe file)', () => {
    const bonus = lines[0].data.map((d) => (d == null ? null : typeof d === 'number' ? Math.round(d * 10) / 10 : d))
    expect(bonus.slice(0, 5)).toEqual([71.4, 81.8, null, null, null])
  })

  it('point final grossi, valeur courte au bout : 60 % et 44 %', () => {
    const endBonus = lines[0].data[6] as Item
    const endPower = lines[1].data[6] as Item
    expect(endBonus.symbolSize).toBe(9)
    expect(lines[0].endLabel!.formatter({ value: endBonus.value })).toBe('60 %')
    expect(lines[1].endLabel!.formatter({ value: endPower.value })).toBe('44 %')
    expect(endBonus.tip).toContain('12 sur 20 (60 %)')
  })

  it('le trait 50 % sur la première courbe seulement', () => {
    expect(lines[0].markLine?.data).toEqual([{ yAxis: 50 }])
    expect(lines[1].markLine).toBeUndefined()
  })

  it('points par match : la part du match, taille = volume (rayon 1,8 + 1,1 × √prises)', () => {
    const first = dots[0].data[0] as Item
    expect(first.value).toEqual([0, (5 / 7) * 100])
    expect(first.symbolSize).toBeCloseTo(2 * pickupRadius(7))
    expect(pickupRadius(16)).toBeCloseTo(1.8 + 4 * 1.1)
    expect(first.tip).toContain('Bonus : équipe 5, adversaire 2 (71,4 %)')
    expect(dots[0].data[4]).toBeNull()
  })

  it('axe par match : rayon de la maquette quel que soit le nombre de matchs (150 : inchangé)', () => {
    const beaucoup = { ...fil, matches: Array.from({ length: 150 }, (_, i) => fil.matches[i % fil.matches.length]) }
    const b = buildResourceFilOption(beaucoup, COLORS, T_FIL) as { series: Series[] }
    const first = b.series.filter((s) => s.type === 'scatter')[0].data[0] as Item
    expect(first.symbolSize).toBeCloseTo(2 * pickupRadius(7))
  })

  it('sous l’axe : l’heure puis la carte ; bande de résultats et encoche de dominance', () => {
    expect(opt.xAxis[0].axisLabel!.formatter('m1', 0)).toBe('{t|19:23}\n{m|Starboard}')
    const band = opt.series.find((s) => s.type === 'custom')!
    const api = (i: number) => ({ value: () => i, coord: () => [100, 50], size: () => [60, 0] })
    const starboard = band.renderItem!(null, api(0)).children
    expect(starboard.map((c) => c.style.fill)).toEqual(['win', 'dom1'])
    const origin = band.renderItem!(null, api(2)).children
    expect(origin.map((c) => c.style.fill)).toEqual(['loss'])
    expect((band.data[0] as Item).tip).toBe('19:23 · Starboard\nVictoire 3–0 · Domination')
  })
})

/**
 * Mode « période » (Séries temporelles › Usages, plan PLAN_TIMESERIES_USAGES_EMPRISE, D12) : sur
 * des dizaines de matchs, pas d'heure ni de carte sous chaque match mais la date du PREMIER match
 * de chaque mois ; aucune mention de couverture (« n matchs, dont m filmés ») ; pas d'encoche de
 * dominance (la maquette n'en dessine pas) ; points plus petits au-delà de 120 matchs.
 */
describe('buildResourceFilOption — mode période', () => {
  const dates = ['2026-07-03', '2026-07-11', '2026-07-28', '2026-08-27', '2026-09-22', '2026-09-22', '2026-09-23']
  const periode = { ...fil, matches: fil.matches.map((m, i) => ({ ...m, startTime: `${dates[i]}T12:00:00Z` })) }
  const axe = { kind: 'period' as const, dateOf: (iso: string) => `${iso.slice(8, 10)}/${iso.slice(5, 7)}` }
  const texte = {
    resourceLabel: (r: string) => T.resources[r].label,
    pctFmt: T.pctFmt,
    pctIntFmt: T.pctIntFmt,
    timeOf: (iso: string) => iso.slice(11, 16),
    outcomeOf: () => null,
    resultOf: () => null,
    dominanceLabel: (d: number) => String(d),
    pointTip: T.fil.pointTip,
    endTip: T.fil.endTip,
    bandTip: T.fil.bandTip,
  }
  type Opt = { series: Series[]; xAxis: Axis[]; graphic?: { style: { text: string } }[] }
  const p = buildResourceFilOption(periode, COLORS, texte, axe) as Opt

  it('la date du premier match de chaque mois sous l’axe, rien sous les autres', () => {
    const f = p.xAxis[0].axisLabel!.formatter
    expect([0, 1, 2, 3, 4, 5, 6].map((i) => f('', i))).toEqual(['{d|03/07}', '', '', '{d|27/08}', '{d|22/09}', '', ''])
  })

  it('aucune mention de couverture sous la bande', () => {
    expect(p.graphic).toBeUndefined()
  })

  it('pas d’encoche de dominance sur la bande de résultats', () => {
    const band = p.series.find((s) => s.type === 'custom')!
    const api = { value: () => 0, coord: () => [100, 50], size: () => [60, 0] }
    expect(band.renderItem!(null, api).children.map((c) => c.style.fill)).toEqual(['win'])
  })

  it('au-delà de 120 matchs, des points plus petits (rayon 0,8 + 0,45 × √prises)', () => {
    const beaucoup = { ...periode, matches: Array.from({ length: 140 }, (_, i) => periode.matches[i % 7]) }
    const b = buildResourceFilOption(beaucoup, COLORS, texte, axe) as Opt
    const first = b.series.filter((s) => s.type === 'scatter')[0].data[0] as Item
    expect(first.symbolSize).toBeCloseTo(2 * (0.8 + Math.sqrt(7) * 0.45))
  })

  it('infobulles d’un match : la date devant l’heure et la carte (une période couvre des mois)', () => {
    const point = p.series.filter((s) => s.type === 'scatter')[0].data[0] as Item
    expect(point.tip).toMatch(/^03\/07 12:00 · Starboard/)
    const band = p.series.find((s) => s.type === 'custom')!
    expect((band.data[0] as Item).tip).toMatch(/^03\/07 12:00 · Starboard/)
  })
  it('le pied de la période (date du mois + bande) tient dans le bas de la grille', () => {
    expect(axisFootProblems(p as never)).toEqual([])
  })

  it('le mode « match » (Escouade) reste le défaut : heure et carte, encoche', () => {
    expect(opt.xAxis[0].axisLabel!.formatter('m1', 0)).toBe('{t|19:23}\n{m|Starboard}')
    expect((opt as Opt).graphic).toBeUndefined()
  })
})

/**
 * Mode « match » COMPACT (tiroir de comparaison de Sessions, maquette `renderFil` avec `cp`) : rien
 * sous l'axe (ni heure, ni carte), graduations 0 / 50 / 100, pied réduit, points plus petits
 * (rayon 1,6 + √prises) ; la bande de résultats et l'encoche de dominance restent.
 */
describe('buildResourceFilOption — mode match compact', () => {
  type Opt = { series: Series[]; xAxis: Axis[]; yAxis: { interval?: number }[]; grid: { bottom: number }[] }
  const c = buildResourceFilOption(fil, COLORS, T_FIL, { kind: 'match', compact: true }) as Opt

  it('rien sous l’axe', () => {
    expect(c.xAxis[0].axisLabel!.formatter('m1', 0)).toBe('')
  })

  it('le pied compact (la bande seule) tient dans le bas de la grille', () => {
    expect(axisFootProblems(c as never)).toEqual([])
  })

  it('graduations tous les 50 %, pied réduit', () => {
    expect(c.yAxis[0].interval).toBe(50)
    expect(c.grid[0].bottom).toBeLessThan((opt as unknown as Opt).grid[0].bottom)
  })

  it('points plus petits (rayon 1,6 + √prises)', () => {
    const first = c.series.filter((s) => s.type === 'scatter')[0].data[0] as Item
    expect(first.symbolSize).toBeCloseTo(2 * (1.6 + Math.sqrt(7)))
  })

  it('bande de résultats et encoche de dominance gardées', () => {
    const band = c.series.find((s) => s.type === 'custom')!
    const api = { value: () => 0, coord: () => [100, 50], size: () => [60, 0] }
    expect(band.renderItem!(null, api).children.map((ch) => ch.style.fill)).toEqual(['win', 'dom1'])
  })
})

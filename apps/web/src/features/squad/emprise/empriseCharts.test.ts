/**
 * empriseCharts.test.ts — « Contrôle des ressources au fil de la session » (22/09) : une courbe
 * cumulée par ressource du bilan, valeur au bout, points par match à la taille de leur volume,
 * trait 50 %, heure et carte sous chaque match, bande de résultats et encoche de dominance.
 */
import { describe, expect, it } from 'vitest'

import { buildHabitOption, buildResourceFilOption, pickupRadius, type EmpriseFilColors, type EmpriseFilText } from './empriseCharts'
import { buildResourceFil, empriseMatchIndex } from './emprise.logic'
import { EMPRISE_2209, HABIT_2209, HISTORY_2209 } from './emprise.fixtures'
import { EMPRISE_TEXT } from './empriseStrings'
import { buildHabitView } from './habit.logic'
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

type HabitSeries = Omit<Series, 'markLine'> & { markLine?: { data: { yAxis: number; lineStyle?: { color?: string; type?: number[] } }[] } }

describe('buildHabitOption — soirée après soirée (22/09)', () => {
  const view = buildHabitView(EMPRISE_2209)
  if (view.kind !== 'chart') throw new Error(view.kind)
  const habit = buildHabitOption(view.resources, view.points, view.medians, { resource: COLORS.resource, parity: 'parity', muted: 'muted', theme: COLORS.theme }, {
    resourceLabel: (r) => T.resources[r].label,
    pctFmt: T.pctFmt,
    pctIntFmt: T.pctIntFmt,
    dateOf: (iso) => `${iso.slice(8, 10)}/${iso.slice(5, 7)}`,
    tonight: T.habit.tonight,
    eveningOf: T.habit.eveningOf,
    pointTip: T.habit.pointTip,
    medianTip: T.habit.medianTip,
    notComparableTip: T.habit.notComparableTip,
  }) as { series: HabitSeries[]; xAxis: Axis }
  const hl = habit.series.filter((s) => s.type === 'line')

  it('une courbe par ressource, à sa couleur ; ce soir grossi, valeur au bout 60 % et 44 %', () => {
    expect(hl.map((s) => s.name)).toEqual(['Bonus', 'Armes spéciales'])
    expect(hl.map((s) => s.lineStyle?.color)).toEqual(['res-powerup', 'res-power_weapon'])
    const tonight = hl[0].data[5] as Item
    expect(tonight.symbolSize).toBe(11)
    expect((hl[0].data[0] as Item).symbolSize).toBe(6)
    expect(hl[0].endLabel!.formatter({ value: tonight.value })).toBe('60 %')
    expect(hl[1].endLabel!.formatter({ value: (hl[1].data[5] as Item).value })).toBe('44 %')
    expect(tonight.tip).toBe('Bonus\nCe soir : 60 %\nMédiane des soirées précédentes : 60 %')
    expect((hl[1].data[0] as Item).tip).toBe('Armes spéciales\nSoirée du 28/07 : 50 %\nMédiane des soirées précédentes : 50 %')
  })

  it('médiane en pointillé fin de la couleur de chaque courbe ; trait 50 % sur la première seulement', () => {
    expect(hl[0].markLine!.data.map((d) => [Math.round(d.yAxis), d.lineStyle?.color, d.lineStyle?.type])).toEqual([
      [60, 'res-powerup', [2, 3]],
      [50, 'parity', [4, 3]],
    ])
    expect(hl[1].markLine!.data.map((d) => [Math.round(d.yAxis), d.lineStyle?.color])).toEqual([[50, 'res-power_weapon']])
  })

  it('dates sous l’axe, « ce soir » en gras ; colonne de ce soir grisée', () => {
    expect(habit.xAxis.axisLabel!.formatter('0', 0)).toBe('{d|28/07}')
    expect(habit.xAxis.axisLabel!.formatter('5', 5)).toBe('{cur|ce soir}')
    const column = habit.series.find((s) => s.type === 'custom')!
    expect(column.data).toEqual([[5, 0]])
  })
})

describe('buildHabitOption — ce soir sans part pour une ressource (constat R4 de la revue L6.1)', () => {
  // Ce soir : des bonus, aucune arme spéciale prise ; les soirées précédentes en ont.
  const block = {
    ...EMPRISE_2209,
    habit: {
      ...HABIT_2209,
      current: { ...HABIT_2209.current, shares: HABIT_2209.current.shares!.filter((s) => s.resource === 'powerup') },
    },
  }
  const view = buildHabitView(block)
  if (view.kind !== 'chart') throw new Error(view.kind)
  const habit = buildHabitOption(view.resources, view.points, view.medians, { resource: COLORS.resource, parity: 'parity', muted: 'muted', theme: COLORS.theme }, {
    resourceLabel: (r) => T.resources[r].label,
    pctFmt: T.pctFmt,
    pctIntFmt: T.pctIntFmt,
    dateOf: (iso) => `${iso.slice(8, 10)}/${iso.slice(5, 7)}`,
    tonight: T.habit.tonight,
    eveningOf: T.habit.eveningOf,
    pointTip: T.habit.pointTip,
    medianTip: T.habit.medianTip,
    notComparableTip: T.habit.notComparableTip,
  }) as { series: (HabitSeries & { endLabel: { show: boolean } })[] }
  const [bonus, armes] = habit.series.filter((s) => s.type === 'line')

  it('armes spéciales : ni point grossi ni valeur au bout — jamais sur la dernière soirée passée', () => {
    expect(view.points[5].current).toBe(true)
    expect(armes.data[5]).toBeNull()
    expect(armes.data.some((d) => d != null && typeof d === 'object' && d.symbolSize === 11)).toBe(false)
    expect(armes.endLabel.show).toBe(false)
  })

  it('bonus : ce soir grossi et valeur au bout, comme d’habitude', () => {
    expect((bonus.data[5] as Item).symbolSize).toBe(11)
    expect(bonus.endLabel.show).toBe(true)
  })
})

describe('buildHabitOption — une soirée d’autres modes que ce soir', () => {
  const previous = HABIT_2209.previous!.map((e, i) => (i === 2 ? { ...e, comparable: false, families: ['Bases', 'Roi de la colline'] } : e))
  const view = buildHabitView({ ...EMPRISE_2209, habit: { ...HABIT_2209, previous } })
  if (view.kind !== 'chart') throw new Error(view.kind)
  const habit = buildHabitOption(view.resources, view.points, view.medians, { resource: COLORS.resource, parity: 'parity', muted: 'muted', theme: COLORS.theme }, {
    resourceLabel: (r) => T.resources[r].label,
    pctFmt: T.pctFmt,
    pctIntFmt: T.pctIntFmt,
    dateOf: (iso) => `${iso.slice(8, 10)}/${iso.slice(5, 7)}`,
    tonight: T.habit.tonight,
    eveningOf: T.habit.eveningOf,
    pointTip: T.habit.pointTip,
    medianTip: T.habit.medianTip,
    notComparableTip: T.habit.notComparableTip,
  }) as { series: HabitSeries[] }

  it('la courbe l’enjambe ; un point gris liseré de la ressource, la raison au survol', () => {
    const [bonusLine] = habit.series.filter((s) => s.type === 'line')
    expect(bonusLine.data[2]).toBeNull()
    const apart = habit.series.filter((s) => s.type === 'scatter')
    expect(apart.map((s) => s.name)).toEqual(['Bonus', 'Armes spéciales'])
    const point = apart[0].data[2] as Item & { itemStyle: { color: string; borderColor: string } }
    expect(point.itemStyle).toMatchObject({ color: 'muted', borderColor: 'res-powerup' })
    expect(point.tip).toBe('Bonus\nSoirée du 27/08 : 75 %\nAutres modes que ce soir (Bases, Roi de la colline) : hors médiane')
    expect(apart[0].data.filter((d) => d != null)).toHaveLength(1)
  })
})

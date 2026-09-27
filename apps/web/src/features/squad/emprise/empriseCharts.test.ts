/**
 * empriseCharts.test.ts — « Contrôle des ressources au fil de la session » (22/09) : une courbe
 * cumulée par ressource du bilan, valeur au bout, points par match à la taille de leur volume,
 * trait 50 %, heure et carte sous chaque match, bande de résultats et encoche de dominance.
 */
import { describe, expect, it } from 'vitest'

import { buildHabitOption, buildResourceFilOption, pickupRadius, type EmpriseFilColors } from './empriseCharts'
import { buildResourceFil } from './emprise.logic'
import { EMPRISE_2209, HABIT_2209, HISTORY_2209 } from './emprise.fixtures'
import { EMPRISE_TEXT } from './empriseStrings'
import { buildHabitView } from './habit.logic'

const T = EMPRISE_TEXT.fr
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

const fil = buildResourceFil(EMPRISE_2209, HISTORY_2209)
const opt = buildResourceFilOption(fil, COLORS, {
  resourceLabel: (r) => T.resources[r].label,
  pctFmt: T.pctFmt,
  pctIntFmt: T.pctIntFmt,
  timeOf: (iso) => iso.slice(11, 16),
  outcomeOf: (m) => (m.outcome ? T.outcomeLower[m.outcome] : null),
  resultOf: (m) => (m.outcome ? `${T.outcome[m.outcome]} ${m.score}` : null),
  dominanceLabel: (d) => (d === 1 ? 'Domination' : String(d)),
  pointTip: T.fil.pointTip,
  endTip: T.fil.endTip,
  bandTip: T.fil.bandTip,
}) as { series: Series[]; xAxis: Axis[] }

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
    expect(first.tip).toContain('Bonus : 5 pour nous, 2 pour eux (71,4 %)')
    expect(dots[0].data[4]).toBeNull()
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

type HabitSeries = Omit<Series, 'markLine'> & { markLine?: { data: { yAxis: number; lineStyle?: { color?: string; type?: number[] } }[] } }

describe('buildHabitOption — soirée après soirée (22/09)', () => {
  const view = buildHabitView(EMPRISE_2209)
  if (view.kind !== 'chart') throw new Error(view.kind)
  const habit = buildHabitOption(view.resources, view.points, view.medians, { resource: COLORS.resource, parity: 'parity', theme: COLORS.theme }, {
    resourceLabel: (r) => T.resources[r].label,
    pctFmt: T.pctFmt,
    pctIntFmt: T.pctIntFmt,
    dateOf: (iso) => `${iso.slice(8, 10)}/${iso.slice(5, 7)}`,
    tonight: T.habit.tonight,
    eveningOf: T.habit.eveningOf,
    pointTip: T.habit.pointTip,
    medianTip: T.habit.medianTip,
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
  const habit = buildHabitOption(view.resources, view.points, view.medians, { resource: COLORS.resource, parity: 'parity', theme: COLORS.theme }, {
    resourceLabel: (r) => T.resources[r].label,
    pctFmt: T.pctFmt,
    pctIntFmt: T.pctIntFmt,
    dateOf: (iso) => `${iso.slice(8, 10)}/${iso.slice(5, 7)}`,
    tonight: T.habit.tonight,
    eveningOf: T.habit.eveningOf,
    pointTip: T.habit.pointTip,
    medianTip: T.habit.medianTip,
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

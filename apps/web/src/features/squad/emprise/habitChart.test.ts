/**
 * habitChart.test.ts — « Contrôle des ressources, par soirée » : barres GROUPÉES, un groupe par
 * soirée et une barre par ressource ; valeur écrite sur les barres de ce soir seulement ; une
 * soirée hors comparaison en barres grisées ; médiane en pointillé ; pied d'axe qui tient.
 */
import { describe, expect, it } from 'vitest'

import { axisFootProblems } from '@/components/charts/axisFoot.check'

import { EMPRISE_2209, HABIT_2209 } from './emprise.fixtures'
import { EMPRISE_TEXT } from './empriseStrings'
import { buildHabitOption, type HabitChartColors } from './habitChart'
import { buildHabitView } from './habit.logic'

const T = EMPRISE_TEXT.fr
const COLORS: HabitChartColors = {
  resource: (r) => `res-${r}`,
  parity: 'parity',
  muted: 'muted',
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

interface Item {
  value: number
  tip: string
  itemStyle: { color: string; borderColor?: string }
  label?: { show: boolean; formatter: () => string }
}
interface BarSeries {
  type: string
  name?: string
  data: (Item | null)[]
  barGap?: string
  markLine?: { data: { yAxis: number; lineStyle?: { color?: string; type?: number[] } }[] }
}
interface Opt {
  series: BarSeries[]
  xAxis: { axisLabel: { formatter: (v: string, i: number) => string } }
}

function build(block = EMPRISE_2209) {
  const view = buildHabitView(block)
  if (view.kind !== 'chart') throw new Error(view.kind)
  const option = buildHabitOption(view.resources, view.points, view.medians, COLORS, {
    resourceLabel: (r) => T.resources[r].label,
    pctFmt: T.pctFmt,
    pctIntFmt: T.pctIntFmt,
    dateOf: (iso) => `${iso.slice(8, 10)}/${iso.slice(5, 7)}`,
    tonight: T.habit.tonight,
    eveningOf: T.habit.eveningOf,
    pointTip: T.habit.pointTip,
    medianTip: T.habit.medianTip,
    notComparableTip: T.habit.notComparableTip,
  })
  return { view, option, opt: option as unknown as Opt }
}

describe('buildHabitOption — barres groupées, soirée par soirée (22/09)', () => {
  const { opt, option } = build()
  const bars = opt.series.filter((s) => s.type === 'bar')

  it('une série de barres par ressource, à sa couleur, dans l’ordre ; plus aucune courbe', () => {
    expect(bars.map((s) => s.name)).toEqual(['Bonus', 'Armes spéciales'])
    expect(bars.map((s) => (s.data[0] as Item).itemStyle.color)).toEqual(['res-powerup', 'res-power_weapon'])
    expect(opt.series.some((s) => s.type === 'line' || s.type === 'scatter')).toBe(false)
  })

  it('la valeur n’est écrite que sur les barres de ce soir : 60 % et 44 %', () => {
    const tonight = bars.map((s) => s.data[5] as Item)
    expect(tonight.map((d) => d.label?.formatter())).toEqual(['60 %', '44 %'])
    for (const s of bars) {
      expect(s.data.slice(0, 5).every((d) => d == null || d.label === undefined)).toBe(true)
    }
    expect((bars[0].data[5] as Item).tip).toBe('Bonus\nCe soir : 60 %\nMédiane des soirées précédentes : 60 %')
    expect((bars[1].data[0] as Item).tip).toBe('Armes spéciales\nSoirée du 28/07 : 50 %\nMédiane des soirées précédentes : 50 %')
  })

  it('médiane en pointillé fin de la couleur de chaque ressource ; trait 50 % sur la première seulement', () => {
    expect(bars[0].markLine!.data.map((d) => [Math.round(d.yAxis), d.lineStyle?.color, d.lineStyle?.type])).toEqual([
      [60, 'res-powerup', [2, 3]],
      [50, 'parity', [4, 3]],
    ])
    expect(bars[1].markLine!.data.map((d) => [Math.round(d.yAxis), d.lineStyle?.color])).toEqual([[50, 'res-power_weapon']])
  })

  it('dates sous l’axe, « ce soir » en gras ; colonne de ce soir grisée', () => {
    expect(opt.xAxis.axisLabel.formatter('0', 0)).toBe('{d|28/07}')
    expect(opt.xAxis.axisLabel.formatter('5', 5)).toBe('{cur|ce soir}')
    const column = opt.series.find((s) => s.type === 'custom')!
    expect(column.data).toEqual([[5, 0]])
  })

  it('le pied d’axe (dates) tient dans le bas de la grille', () => {
    expect(axisFootProblems(option as never)).toEqual([])
  })
})

describe('buildHabitOption — ce soir sans part pour une ressource (constat R4 de la revue L6.1)', () => {
  // Ce soir : des bonus, aucune arme spéciale prise ; les soirées précédentes en ont.
  const { view, opt } = build({
    ...EMPRISE_2209,
    habit: {
      ...HABIT_2209,
      current: { ...HABIT_2209.current, shares: HABIT_2209.current.shares!.filter((s) => s.resource === 'powerup') },
    },
  })
  const [bonus, armes] = opt.series.filter((s) => s.type === 'bar')

  it('armes spéciales : aucune barre ni valeur ce soir — jamais la dernière soirée passée', () => {
    expect(view.points[5].current).toBe(true)
    expect(armes.data[5]).toBeNull()
    expect(armes.data.some((d) => d?.label !== undefined)).toBe(false)
  })

  it('bonus : la valeur de ce soir est écrite, comme d’habitude', () => {
    expect((bonus.data[5] as Item).label?.show).toBe(true)
  })
})

describe('buildHabitOption — une soirée d’autres modes que ce soir', () => {
  const previous = HABIT_2209.previous!.map((e, i) => (i === 2 ? { ...e, comparable: false, families: ['Bases', 'Roi de la colline'] } : e))
  const { opt } = build({ ...EMPRISE_2209, habit: { ...HABIT_2209, previous } })
  const [bonus, armes] = opt.series.filter((s) => s.type === 'bar')

  it('barre GRISE liserée de la couleur de la ressource, la raison au survol, hors médiane', () => {
    const bar = bonus.data[2] as Item
    expect(bar.itemStyle).toMatchObject({ color: 'muted', borderColor: 'res-powerup' })
    expect(bar.tip).toBe('Bonus\nSoirée du 27/08 : 75 %\nAutres modes que ce soir (Bases, Roi de la colline) : hors médiane')
    expect((armes.data[2] as Item).itemStyle).toMatchObject({ color: 'muted', borderColor: 'res-power_weapon' })
    // Les autres soirées gardent la couleur de la ressource.
    expect((bonus.data[1] as Item).itemStyle.color).toBe('res-powerup')
  })
})

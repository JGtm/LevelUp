import { describe, expect, it } from 'vitest'

import { buildEveningsOption } from './eveningsChart'
import { buildFilOption, shortMap, volumeRadius, type ObjectifChartColors } from './objectifCharts'
import { buildEveningsView, buildSessionFil } from './objectif.logic'
import { block0709, history0709, history0709Evenings } from './objectif.fixtures'
import { OBJECTIF_TEXT } from './objectifStrings'

const T = OBJECTIF_TEXT.fr
const COLORS: ObjectifChartColors = {
  roles: { take: 'take', defend: 'defend', hold: 'hold' },
  win: 'win',
  loss: 'loss',
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

type Series = { type: string; name?: string; data: unknown[]; markLine?: { data: { yAxis: number }[] }; endLabel?: { formatter: (p: { value: unknown }) => string } }
type Axis = { axisLabel: { formatter: (v: string, i: number) => string } }

describe('buildFilOption — au fil de la session (07/09)', () => {
  const fil = buildSessionFil(block0709(), history0709())
  const opt = buildFilOption(fil, COLORS, {
    roles: T.roles,
    pctFmt: (v) => T.pctFmt(v),
    countFmt: (v, d) => (d ? T.durationFmt(v) : String(v)),
    timeOf: (iso) => iso.slice(11, 16),
    familyLabel: (f) => (f === 'ctf' ? 'Drapeau' : 'Bases'),
    contextOf: (m) => T.fil.contextFmt(m.family, m.outcome ? T.outcome[m.outcome] : null, m.score),
    dominanceLabel: (d) => `d${d}`,
    pointTip: T.fil.pointTip,
    bandTip: T.fil.bandTip,
  }) as { series: Series[]; xAxis: Axis[] }

  it('trois courbes de rôle, leurs parts par match, puis la bande de résultats', () => {
    expect(opt.series.map((s) => s.type)).toEqual(['line', 'scatter', 'line', 'scatter', 'line', 'scatter', 'custom'])
    expect(opt.series[0].markLine?.data).toEqual([{ yAxis: 50 }])
  })

  it('valeur au bout de la courbe (arrondie) et point final grossi', () => {
    const take = opt.series[0]
    const last = take.data[take.data.length - 1] as { value: number; symbolSize: number }
    expect(Math.round(last.value * 10) / 10).toBe(39.0)
    expect(last.symbolSize).toBe(9)
    expect(take.endLabel?.formatter({ value: last.value })).toBe('39 %')
  })

  it('taille des points = volume du lobby ; heure sous l’axe, carte et mode sous la bande', () => {
    const pts = opt.series[1].data as { symbolSize: number }[]
    // b3 (64 actions) est le plus gros volume de Prendre.
    expect(pts[2].symbolSize).toBeCloseTo(14, 6)
    expect(opt.xAxis[0].axisLabel.formatter('', 0)).toBe('19:26')
    expect(opt.xAxis[1].axisLabel.formatter('', 0)).toBe('{map|Banished N.}\n{mode|Bases}')
  })

  it('utilitaires de forme', () => {
    expect(shortMap('Banished Narrows')).toBe('Banished N.')
    expect(shortMap('Origin')).toBe('Origin')
    expect(volumeRadius(64, 64)).toBe(7)
    expect(volumeRadius(0, 0)).toBe(2)
  })
})

describe('buildEveningsOption — soirée après soirée (07/09)', () => {
  const view = buildEveningsView(history0709Evenings())
  if (view.kind !== 'chart') throw new Error('attendu : graphe')
  const opt = buildEveningsOption(view.points, view.medians, COLORS, {
    roles: T.roles,
    pctFmt: T.pctFmt,
    tonight: T.evenings.tonight,
    dateOf: (iso) => `${iso.slice(8, 10)}/${iso.slice(5, 7)}`,
    outOfFmt: T.evenings.outOfFmt,
    mixOf: () => '4 B · 3 D',
    pointTip: T.evenings.pointTip,
    bandTip: T.evenings.bandTip,
    eveningOf: T.evenings.eveningOf,
    medianTip: (r, v) => `${r} ${v}`,
  }) as { series: Series[]; xAxis: Axis[] }

  it('trois courbes, médiane des précédentes en pointillé de leur couleur, trait 50 % sur la première', () => {
    const lines = opt.series.filter((s) => s.type === 'line')
    expect(lines).toHaveLength(3)
    expect(lines[0].markLine?.data.map((d) => d.yAxis)).toEqual([50, 50])
    expect(lines[1].markLine?.data.map((d) => Math.round(d.yAxis * 100) / 100)).toEqual([51.1])
  })

  it('ce soir à droite, sous chaque soirée « x sur y » et les modes', () => {
    expect(opt.xAxis[0].axisLabel.formatter('', 10)).toBe('{cur|ce soir}')
    expect(opt.xAxis[0].axisLabel.formatter('', 0)).toBe('{d|06/04}')
    expect(opt.xAxis[1].axisLabel.formatter('', 10)).toBe('{n|1 sur 7}\n{m|4 B · 3 D}')
    expect(opt.series.filter((s) => s.type === 'custom')).toHaveLength(2)
  })
})

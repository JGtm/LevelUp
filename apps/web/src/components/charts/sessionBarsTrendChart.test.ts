/**
 * La frise partagée « une soirée, un bâton » — la grammaire tenue, quel que soit le
 * nombre de séries.
 *
 * Ce que ces tests cadenassent : un seul axe Y en pourcents (deux seraient une faute de
 * lecture), DEUX séries de bâtons et DEUX repères tiretés sur le même axe (le cas des
 * Séries temporelles, D22-3), le bâton creux d'un échantillon faible, le rang de volumes
 * optionnel sous l'axe, et l'option vide sans soirée.
 */
import { describe, expect, it } from 'vitest'

import { buildSessionBarsTrendOption, shortSessionLabel } from './sessionBarsTrendChart'

const LABELS = ['12/09', '14/09', '20/09']

function deuxSeries() {
  return buildSessionBarsTrendOption({
    labels: LABELS,
    yAxisLabel: 'Part de la soirée',
    series: [
      {
        name: 'Je suis couvert',
        color: '#111111',
        valuesPct: [54.2, 48, null],
        hollow: [false, true, false],
        usual: { valuePct: 48.4, label: 'habituel 48 %', color: '#111111' },
      },
      {
        name: 'Je riposte',
        color: '#222222',
        valuesPct: [29.1, 26, 24],
        usual: { valuePct: 25, label: 'parité 25 %', color: '#222222' },
      },
    ],
    tooltipLines: (i) => [`mes morts : ${[12, 9, 14][i]}`],
  }) as Record<string, unknown>
}

describe('buildSessionBarsTrendOption', () => {
  it('rend UN SEUL axe Y, en pourcents et ancré à zéro', () => {
    const y = deuxSeries().yAxis as Record<string, unknown>
    expect(Array.isArray(y)).toBe(false)
    expect(y.min).toBe(0)
    expect((y.axisLabel as { formatter: string }).formatter).toBe('{value} %')
  })

  it('pose DEUX séries de bâtons et DEUX repères tiretés sur le même axe', () => {
    const series = deuxSeries().series as Record<string, unknown>[]
    const barres = series.filter((s) => s.type === 'bar')
    expect(barres).toHaveLength(2)
    expect(barres.every((s) => s.barMaxWidth === 18)).toBe(true)
    const reperes = barres.map(
      (s) => s.markLine as { lineStyle: { type: string; color: string }; data: { yAxis: number }[] },
    )
    expect(reperes.map((m) => m.lineStyle.type)).toEqual(['dashed', 'dashed'])
    expect(reperes.map((m) => m.data[0].yAxis)).toEqual([48.4, 25])
    // Chaque repère porte la couleur de SA série : deux traits gris ne se rattacheraient
    // à rien sur un graphe qui en porte deux.
    expect(reperes.map((m) => m.lineStyle.color)).toEqual(['#111111', '#222222'])
  })

  it('rend un bâton CREUX pour une soirée à échantillon faible, jamais un trou', () => {
    const barres = (deuxSeries().series as Record<string, unknown>[]).filter((s) => s.type === 'bar')
    const data = barres[0].data as (number | { value: number; itemStyle: { color: string; borderColor: string } } | null)[]
    expect(data[0]).toBe(54.2)
    expect(data[1]).toEqual({
      value: 48,
      itemStyle: { color: 'transparent', borderColor: '#111111', borderWidth: 1.5 },
    })
    expect(data[2]).toBeNull()
  })

  it('nomme chaque série ET chaque tendance dans la légende, les courbes en dernier', () => {
    const option = buildSessionBarsTrendOption({
      labels: LABELS,
      yAxisLabel: 'y',
      series: [
        {
          name: 'au-dessus',
          color: '#1',
          valuesPct: [1, 2, 3],
          trend: { valuesPct: [1, 1.5, 2], label: 'tendance', color: '#3' },
        },
        { name: 'en dessous', color: '#2', valuesPct: [null, null, null] },
      ],
    }) as Record<string, unknown>
    expect((option.legend as { data: string[] }).data).toEqual([
      'au-dessus',
      'en dessous',
      'tendance',
    ])
    const series = option.series as Record<string, unknown>[]
    expect(series.map((s) => s.type)).toEqual(['bar', 'bar', 'line'])
    expect((series[2].lineStyle as { width: number }).width).toBe(2)
    expect(series[2].smooth).toBe(false)
  })

  it('pose les VOLUMES en second rang d’étiquettes quand on les lui donne, sinon un seul axe', () => {
    const avec = buildSessionBarsTrendOption({
      labels: LABELS,
      yAxisLabel: 'y',
      series: [{ name: 'a', color: '#1', valuesPct: [1, 2, 3] }],
      volumeAxis: { label: 'morts mesurées', values: ['84', '61', '73'] },
    }) as Record<string, unknown>
    const x = avec.xAxis as Record<string, unknown>[]
    expect(x).toHaveLength(2)
    expect(x[1].data).toEqual(['84', '61', '73'])
    expect((x[1].axisLine as { show: boolean }).show).toBe(false)
    expect((deuxSeries().xAxis as unknown[])).toHaveLength(1)
  })

  it('rend une option VIDE sans soirée : pas d’axes fantômes', () => {
    const vide = buildSessionBarsTrendOption({ labels: [], series: [], yAxisLabel: 'y' }) as Record<string, unknown>
    expect(vide.series).toBeUndefined()
    expect(vide.xAxis).toBeUndefined()
  })
})

/**
 * Le MODE ÉCART (3.B, D23-3 du 2026-09-22) : la frise soustrait à chaque série SON
 * repère. Ce qui est cadenassé ici, c'est le basculement complet de la grammaire — les
 * valeurs, l'origine de l'axe, l'unique ligne zéro, la disparition des tiretés, le style
 * de la tendance et les deux lectures de l'infobulle.
 */
function enEcart() {
  return buildSessionBarsTrendOption({
    labels: LABELS,
    yAxisLabel: 'Écart (points)',
    baseline: { label: 'habituel 48 % · parité 25 %', deltaUnit: 'pts' },
    hollowLegend: { label: 'Échantillon faible', color: '#111111' },
    series: [
      {
        name: 'Je suis couvert',
        color: '#111111',
        valuesPct: [54.2, 48, null],
        hollow: [false, true, false],
        usual: { valuePct: 48, label: 'habituel 48 %' },
        trend: { valuesPct: [null, 50, 52], label: 'Tendance — Je suis couvert', color: '#111111' },
      },
      {
        name: 'Je riposte',
        color: '#222222',
        valuesPct: [29.1, 26, 24],
        usual: { valuePct: 25, label: 'parité 25 %' },
      },
    ],
  }) as Record<string, unknown>
}

describe('buildSessionBarsTrendOption — mode écart', () => {
  it('trace chaque série en ÉCART à son propre repère, bâton creux compris', () => {
    const barres = (enEcart().series as Record<string, unknown>[]).filter((s) => s.type === 'bar')
    expect(barres[0].data).toEqual([
      6.2,
      { value: 0, itemStyle: { color: 'transparent', borderColor: '#111111', borderWidth: 1.5 } },
      null,
    ])
    expect(barres[1].data).toEqual([4.1, 1, -1])
  })

  it('pose UNE SEULE ligne, à zéro et en trait PLEIN — plus aucun tireté de série', () => {
    const barres = (enEcart().series as Record<string, unknown>[]).filter((s) => s.type === 'bar')
    const lignes = barres.filter((s) => s.markLine != null)
    expect(lignes).toHaveLength(1)
    const m = lignes[0].markLine as {
      lineStyle: { type: string }
      label: { formatter: string }
      data: { yAxis: number }[]
    }
    expect(m.lineStyle.type).toBe('solid')
    expect(m.data[0].yAxis).toBe(0)
    // L'étiquette NOMME les deux repères qu'elle remplace : sans elle, plus de référence.
    expect(m.label.formatter).toBe('habituel 48 % · parité 25 %')
  })

  it('libère l’axe sous zéro et le compte en points signés, plus en parts', () => {
    const y = enEcart().yAxis as Record<string, unknown>
    expect(y.min).toBeUndefined()
    const fmt = (y.axisLabel as { formatter: (v: number) => string }).formatter
    expect([fmt(10), fmt(0), fmt(-10)]).toEqual(['+10', '0', '-10'])
  })

  it('décale la tendance comme ses bâtons et la trace en pointillé fin, sans marqueur', () => {
    const ligne = (enEcart().series as Record<string, unknown>[]).find((s) => s.type === 'line')
    expect(ligne?.data).toEqual([null, 2, 4])
    expect(ligne?.lineStyle).toEqual({
      width: 1.6,
      type: 'dashed',
      color: '#111111',
      opacity: 0.75,
    })
    expect(ligne?.symbol).toBe('none')
  })

  it('porte les DEUX lectures en infobulle : la valeur absolue et l’écart signé', () => {
    const tooltip = enEcart().tooltip as { formatter: (p: unknown) => string }
    const rendu = tooltip.formatter([
      { axisValue: '12/09', dataIndex: 0, marker: '<m>', seriesName: 'Je suis couvert', value: 6.2 },
      { axisValue: '12/09', dataIndex: 0, marker: '<m>', seriesName: 'Je riposte', value: -1 },
    ])
    expect(rendu).toContain('Je suis couvert : 54.2 % (+6.2 pts)')
    expect(rendu).toContain('Je riposte : 24 % (-1 pts)')
  })

  it('nomme le bâton creux en légende par un TÉMOIN sans donnée', () => {
    const option = enEcart()
    expect((option.legend as { data: string[] }).data).toEqual([
      'Je suis couvert',
      'Je riposte',
      'Tendance — Je suis couvert',
      'Échantillon faible',
    ])
    const temoin = (option.series as Record<string, unknown>[]).at(-1)
    expect(temoin?.name).toBe('Échantillon faible')
    expect(temoin?.data).toEqual([])
    expect(temoin?.itemStyle).toEqual({
      color: 'transparent',
      borderColor: '#111111',
      borderWidth: 1.5,
    })
  })
})

describe('shortSessionLabel', () => {
  it('réduit un libellé de session à sa date, et laisse intact ce qui n’a pas d’espace', () => {
    expect(shortSessionLabel('13/10/2025 22:27–22:46 (3)')).toBe('13/10/2025')
    expect(shortSessionLabel('2026-S18')).toBe('2026-S18')
  })
})

/**
 * FirstBloodLanes — structure de l'option ECharts (echarts-for-react mocké, comme
 * OutcomeSequenceTape) : nombre et type des séries, ordre des lanes, décalages des
 * nuages, géométrie de la barre d'avance, tooltips FR/EN et état vide.
 * Le calcul (médianes, tri, formats) est testé à part : firstBloodLanesModel.test.ts.
 */
import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { FirstBloodLanes } from './FirstBloodLanes'
import type { FirstBloodPlayerSeries } from './firstBloodLanesModel'

// Couleurs déterministes : en jsdom les CSS vars de palette ne sont pas résolues.
vi.mock('@/lib/accessibility', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/accessibility')>()),
  resolveToken: (token: string) => `tok:${token}`,
}))

const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: Record<string, unknown>) => {
    captured.push(props)
    return <div data-testid="lanes-stub" />
  },
}))

interface SeriesLike {
  type: string
  z?: number
  data: Array<Record<string, unknown>>
  symbolOffset?: [number, number]
  symbolSize?: number
  itemStyle?: Record<string, unknown>
  tooltip?: { formatter: (p: unknown) => string }
  renderItem?: (params: unknown, api: unknown) => Record<string, unknown>
}

interface OptionLike {
  series: SeriesLike[]
  xAxis: Record<string, unknown>
  yAxis: {
    type: string
    data: string[]
    inverse: boolean
    axisLabel: { formatter: (v: string, i: number) => string; margin: number; align: string }
  }
  tooltip: { trigger: string }
}

const DATA: FirstBloodPlayerSeries[] = [
  {
    player: 'Slow',
    matches: [
      { matchId: 'm1', firstKillSec: 80, firstDeathSec: 40 },
      { matchId: 'm2', firstKillSec: 100, firstDeathSec: 60 },
    ],
  },
  {
    player: 'Fast',
    matches: [
      { matchId: 'm1', firstKillSec: 20, firstDeathSec: 50 },
      { matchId: 'm2', firstKillSec: 40, firstDeathSec: 70 },
      { matchId: 'm3', firstKillSec: null, firstDeathSec: null },
    ],
  },
]

async function renderChart(props: Partial<Parameters<typeof FirstBloodLanes>[0]> = {}) {
  captured.length = 0
  render(<FirstBloodLanes data={DATA} {...props} />)
  await screen.findByTestId('lanes-stub')
  const last = captured[captured.length - 1]
  return { option: last.option as OptionLike, props: last }
}

describe('FirstBloodLanes — structure de l’option', () => {
  beforeEach(() => {
    captured.length = 0
  })

  it('produit 5 séries : 1 custom (barre d’avance) + 4 scatter (nuages + médianes)', async () => {
    const { option } = await renderChart()
    expect(option.series).toHaveLength(5)
    expect(option.series.map((s) => s.type)).toEqual([
      'custom',
      'scatter',
      'scatter',
      'scatter',
      'scatter',
    ])
    // Ordre de dessin : barre au fond, nuages, médianes au-dessus.
    expect(option.series.map((s) => s.z)).toEqual([1, 2, 2, 3, 3])
  })

  it('trie les lanes par médiane du premier frag et inverse l’axe (plus rapide en haut)', async () => {
    const { option } = await renderChart()
    expect(option.yAxis.type).toBe('category')
    expect(option.yAxis.data).toEqual(['Fast', 'Slow'])
    expect(option.yAxis.inverse).toBe(true)
    // La colonne de libellés est ancrée À GAUCHE du grid (rien dans le tracé).
    expect(option.yAxis.axisLabel.align).toBe('left')
    expect(option.yAxis.axisLabel.margin).toBe(130)
  })

  it('cadre l’axe X sur [0, maxSec] avec un tick par minute', async () => {
    const { option } = await renderChart({ maxSec: 240 })
    expect(option.xAxis).toMatchObject({ type: 'value', min: 0, max: 240, interval: 60 })
  })

  it('décale les nuages de ±14 px et exclut les matchs sans événement', async () => {
    const { option } = await renderChart()
    const [, kills, deaths] = option.series
    expect(kills.symbolOffset).toEqual([0, -14])
    expect(deaths.symbolOffset).toEqual([0, 14])
    expect(kills.symbolSize).toBe(6)
    // 2 joueurs × 2 matchs exploitables (le 3e match de Fast est null/null).
    expect(kills.data).toHaveLength(4)
    expect(deaths.data).toHaveLength(4)
    // Lane index 0 = Fast (trié en tête).
    expect(kills.data[0].value).toEqual([20, 0])
  })

  it('pose un marqueur de médiane par lane, cerclé de la couleur de fond', async () => {
    const { option } = await renderChart()
    const [, , , medKills, medDeaths] = option.series
    expect(medKills.symbolSize).toBe(16)
    expect(medKills.itemStyle).toMatchObject({ color: 'tok:outcome-win', borderWidth: 2 })
    expect(medDeaths.itemStyle).toMatchObject({ color: 'tok:outcome-loss', borderWidth: 2 })
    expect(medKills.data.map((d) => d.value)).toEqual([
      [30, 0],
      [90, 1],
    ])
    expect(medDeaths.data.map((d) => d.value)).toEqual([
      [60, 0],
      [50, 1],
    ])
  })
})

describe('FirstBloodLanes — barre d’avance', () => {
  it('dessine un rectangle arrondi de 8 px entre les deux médianes', async () => {
    const { option } = await renderChart()
    const gap = option.series[0]
    const api = { coord: ([x, y]: number[]) => [x * 2, 100 + y * 54] }
    const el = gap.renderItem?.({ dataIndex: 0 }, api) as {
      type: string
      shape: { x: number; y: number; width: number; height: number; r: number }
      style: { fill: string; opacity: number }
    }
    // Fast : médianes 30 → 60 s, soit 60 → 120 px avec ce stub d'API.
    expect(el.type).toBe('rect')
    expect(el.shape).toMatchObject({ x: 60, width: 60, height: 8, r: 4 })
    expect(el.shape.y).toBe(100 - 4)
    expect(el.style).toMatchObject({ fill: 'tok:outcome-win', opacity: 0.32 })
  })

  it('vire au rouge quand la première mort précède le premier frag', async () => {
    const { option } = await renderChart()
    const gap = option.series[0]
    const api = { coord: ([x, y]: number[]) => [x * 2, 100 + y * 54] }
    // Slow : médiane mort 50 s < médiane frag 90 s → écart négatif.
    const el = gap.renderItem?.({ dataIndex: 1 }, api) as {
      shape: { x: number; width: number }
      style: { fill: string }
    }
    expect(el.style.fill).toBe('tok:outcome-loss')
    expect(el.shape).toMatchObject({ x: 100, width: 80 })
  })

  it('n’émet aucune barre pour une lane sans les deux médianes', async () => {
    captured.length = 0
    render(
      <FirstBloodLanes
        data={[
          { player: 'Silent', matches: [{ matchId: 'a', firstKillSec: null, firstDeathSec: 33 }] },
        ]}
      />,
    )
    await screen.findByTestId('lanes-stub')
    const option = captured[captured.length - 1].option as OptionLike
    expect(option.series[0].data).toHaveLength(0)
  })
})

describe('FirstBloodLanes — libellés et tooltips', () => {
  it('compose la colonne de gauche : pseudo, médianes colorées, avance signée', async () => {
    const { option } = await renderChart()
    const label = option.yAxis.axisLabel.formatter('Fast', 0)
    const lines = label.split('\n')
    expect(lines[0]).toBe('{gt|Fast}')
    expect(lines[1]).toBe('{med|méd. }{kill|30s}{med| → }{death|1m00}')
    expect(lines[2]).toBe("{gapPos|+30s d'avance}")
    // Lane en retard → style rouge et signe moins typographique.
    expect(option.yAxis.axisLabel.formatter('Slow', 1)).toContain("{gapNeg|−40s d'avance}")
  })

  it('tooltip item : match, médiane (couverture n/total) et fenêtre d’avance', async () => {
    const { option } = await renderChart()
    expect(option.tooltip.trigger).toBe('item')
    const [gapS, killS, , medKillS] = option.series

    expect(killS.tooltip?.formatter({ data: killS.data[0] })).toBe(
      'Fast · match m1 · premier frag 20s',
    )
    // Fast : 2 premiers frags exploitables sur 3 matchs.
    expect(medKillS.tooltip?.formatter({ data: medKillS.data[0] })).toContain(
      'médiane premier frag 30s (2/3 matchs)',
    )
    expect(gapS.tooltip?.formatter({ data: gapS.data[0] })).toContain(
      "fenêtre d'avance médiane : +30s",
    )
  })

  it('bascule en anglais avec la locale EN', async () => {
    const { option } = await renderChart({ locale: 'en' })
    expect(option.yAxis.axisLabel.formatter('Fast', 0)).toContain('{gapPos|+30s ahead}')
    expect(option.series[1].tooltip?.formatter({ data: option.series[1].data[0] })).toBe(
      'Fast · match m1 · first kill 20s',
    )
  })

  it('échappe le HTML des données non constantes (pseudo, identifiant de match)', async () => {
    captured.length = 0
    render(
      <FirstBloodLanes
        data={[
          {
            player: '<img src=x>',
            matches: [{ matchId: '"><b>', firstKillSec: 10, firstDeathSec: 20 }],
          },
        ]}
      />,
    )
    await screen.findByTestId('lanes-stub')
    const option = captured[captured.length - 1].option as OptionLike
    const html = option.series[1].tooltip?.formatter({ data: option.series[1].data[0] }) ?? ''
    expect(html).not.toContain('<img')
    expect(html).toContain('&lt;img src=x&gt;')
  })
})

describe('FirstBloodLanes — états', () => {
  it('rend le titre par défaut du manifest', async () => {
    await renderChart()
    expect(screen.getByText('Premier frag / première mort')).toBeTruthy()
  })

  it('affiche l’état vide quand aucun événement n’est exploitable', () => {
    render(
      <FirstBloodLanes
        data={[
          { player: 'Ghost', matches: [{ matchId: 'a', firstKillSec: null, firstDeathSec: null }] },
        ]}
      />,
    )
    expect(screen.getByTestId('chart-card-empty')).toBeTruthy()
    expect(screen.getByText(/Aucun premier frag/)).toBeTruthy()
  })

  it('dimensionne la carte sur le nombre de lanes (54 px par bande)', async () => {
    const { props } = await renderChart()
    // 2 lanes × 54 + marges du grid (8 + 28) = 144.
    expect((props.style as { height: number }).height).toBe(144)
  })
})

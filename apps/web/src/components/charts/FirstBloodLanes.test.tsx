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
      { matchId: 'm1', firstKillSec: 80, firstDeathSec: 40, mapUI: 'Bazaar', modeUI: 'CTF', startTime: '2026-04-18T12:00:00Z' },
      { matchId: 'm2', firstKillSec: 100, firstDeathSec: 60, mapUI: 'Bazaar', modeUI: 'CTF', startTime: '2026-04-19T12:00:00Z' },
      // 3e match AJOUTÉ pile sur la médiane 2-points pré-existante (90/50) :
      // porte le total à 3 (MIN_MATCHES_FOR_CLOUD) sans déplacer médiane ni
      // écart — les assertions numériques ci-dessous (gap, labels) restent
      // valides inchangées.
      { matchId: 'm3', firstKillSec: 90, firstDeathSec: 50, mapUI: 'Bazaar', modeUI: 'CTF', startTime: '2026-04-20T12:00:00Z' },
    ],
  },
  {
    player: 'Fast',
    matches: [
      { matchId: 'm1', firstKillSec: 20, firstDeathSec: 50, mapUI: 'Aquarius', modeUI: 'Slayer', startTime: '2026-04-19T12:00:00Z' },
      { matchId: 'm2', firstKillSec: 40, firstDeathSec: 70, mapUI: 'Aquarius', modeUI: 'Slayer', startTime: '2026-04-19T13:00:00Z' },
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
    expect(kills.symbolSize).toBe(8)
    // Fast (3 matchs, 1 sans événement → 2 exploitables) + Slow (3 matchs,
    // tous exploitables) = 5. Aucune lane n'est sous MIN_MATCHES_FOR_CLOUD ici.
    expect(kills.data).toHaveLength(5)
    expect(deaths.data).toHaveLength(5)
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

  it('tooltip item : carte, mode, date et fenêtre d’avance (DEC-4 — jamais l’uuid)', async () => {
    const { option } = await renderChart()
    expect(option.tooltip.trigger).toBe('item')
    const [gapS, killS, , medKillS] = option.series

    // Noon UTC choisi pour la date afin que le rendu Intl reste sur le même
    // jour calendaire quel que soit le fuseau de la machine de test.
    expect(killS.tooltip?.formatter({ data: killS.data[0] })).toBe(
      'Fast · Aquarius · Slayer · 19 avr. 2026 · premier frag 20s',
    )
    expect(killS.tooltip?.formatter({ data: killS.data[0] })).not.toContain('m1')
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
      'Fast · Aquarius · Slayer · Apr 19, 2026 · first kill 20s',
    )
  })

  it('dégrade proprement quand carte/mode manquent sur un point : jamais l’uuid, au moins la date', async () => {
    captured.length = 0
    render(
      <FirstBloodLanes
        data={[
          {
            player: 'NoMeta',
            matches: [
              { matchId: 'a', firstKillSec: 10, firstDeathSec: 20, startTime: '2026-04-19T12:00:00Z' },
              { matchId: 'b', firstKillSec: 15, firstDeathSec: 25, startTime: '2026-04-20T12:00:00Z' },
              { matchId: 'c', firstKillSec: 12, firstDeathSec: 22, startTime: '2026-04-21T12:00:00Z' },
            ],
          },
        ]}
      />,
    )
    await screen.findByTestId('lanes-stub')
    const option = captured[captured.length - 1].option as OptionLike
    const html = option.series[1].tooltip?.formatter({ data: option.series[1].data[0] }) ?? ''
    expect(html).not.toContain('undefined')
    expect(html).not.toMatch(/match [a-z]\b/) // pas l'identifiant de match brut
    expect(html).toContain('—') // placeholder carte/mode, jamais une clé brute
    expect(html).toContain('19 avr. 2026') // la date, elle, reste toujours affichée
  })

  it('échappe le HTML des données non constantes (pseudo, carte, mode)', async () => {
    captured.length = 0
    render(
      <FirstBloodLanes
        data={[
          {
            player: '<img src=x>',
            matches: [
              {
                matchId: 'a',
                firstKillSec: 10,
                firstDeathSec: 20,
                mapUI: '<b>Map</b>',
                modeUI: '<i>Mode</i>',
                startTime: '2026-04-19T12:00:00Z',
              },
              { matchId: 'b', firstKillSec: 15, firstDeathSec: 25 },
              { matchId: 'c', firstKillSec: 12, firstDeathSec: 22 },
            ],
          },
        ]}
      />,
    )
    await screen.findByTestId('lanes-stub')
    const option = captured[captured.length - 1].option as OptionLike
    const html = option.series[1].tooltip?.formatter({ data: option.series[1].data[0] }) ?? ''
    expect(html).not.toContain('<img')
    expect(html).not.toContain('<b>Map</b>')
    expect(html).not.toContain('<i>Mode</i>')
    expect(html).toContain('&lt;img src=x&gt;')
    expect(html).toContain('&lt;b&gt;Map&lt;/b&gt;')
    expect(html).toContain('&lt;i&gt;Mode&lt;/i&gt;')
  })
})

describe('FirstBloodLanes — nuage supprimé à faible N (lisibilité, retour utilisateur 2026-08-29)', () => {
  it('ne dessine aucun point de nuage pour une lane à 1 ou 2 matchs', async () => {
    captured.length = 0
    render(
      <FirstBloodLanes
        data={[
          { player: 'OneMatch', matches: [{ matchId: 'a', firstKillSec: 10, firstDeathSec: 20 }] },
          {
            player: 'TwoMatches',
            matches: [
              { matchId: 'b', firstKillSec: 10, firstDeathSec: 20 },
              { matchId: 'c', firstKillSec: 15, firstDeathSec: 25 },
            ],
          },
        ]}
      />,
    )
    await screen.findByTestId('lanes-stub')
    const option = captured[captured.length - 1].option as OptionLike
    const [, kills, deaths] = option.series
    expect(kills.data).toHaveLength(0)
    expect(deaths.data).toHaveLength(0)
  })

  it('dessine le nuage dès 3 matchs ; médiane et barre d’avance restent, elles, toujours présentes', async () => {
    captured.length = 0
    render(
      <FirstBloodLanes
        data={[
          {
            player: 'ThreeMatches',
            matches: [
              { matchId: 'a', firstKillSec: 10, firstDeathSec: 20 },
              { matchId: 'b', firstKillSec: 15, firstDeathSec: 25 },
              { matchId: 'c', firstKillSec: 20, firstDeathSec: 30 },
            ],
          },
        ]}
      />,
    )
    await screen.findByTestId('lanes-stub')
    const option = captured[captured.length - 1].option as OptionLike
    const [gap, kills, deaths, medKills, medDeaths] = option.series
    expect(kills.data).toHaveLength(3)
    expect(deaths.data).toHaveLength(3)
    // Médiane et barre d'avance : un point par lane, indépendant du seuil nuage.
    expect(medKills.data).toHaveLength(1)
    expect(medDeaths.data).toHaveLength(1)
    expect(gap.data).toHaveLength(1)
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

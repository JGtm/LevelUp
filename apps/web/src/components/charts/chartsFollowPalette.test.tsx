/**
 * Les deux graphes qui montent `LazyECharts` sans passer par `ChartCard`
 * (`CumulativeFragGapChart`, `OutcomeSequenceTape`) suivent un changement de palette
 * d'accessibilité SANS remontage : les couleurs de l'option passée à ECharts sont recalculées
 * quand `applyPalette` réécrit les variables de `:root`.
 *
 * Avant l'étape 7.3, aucun des deux n'écoutait la palette : leurs couleurs restaient celles de
 * l'ancienne palette jusqu'au prochain montage (audit TEC-1).
 */
import { act, render, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { _resetActivePalette, applyPalette } from '@/lib/accessibility/applyPalette'
import { defaultPalette } from '@/lib/accessibility/palettes/default'
import { okabePalette } from '@/lib/accessibility/palettes/okabe-ito'

import { CumulativeFragGapChart } from './CumulativeFragGapChart'
import { OutcomeSequenceTape } from './OutcomeSequenceTape'

// Le montage réel d'ECharts (canvas) est remplacé par un témoin qui garde la dernière option reçue.
const received: Array<{ option: unknown }> = []
vi.mock('echarts-for-react', () => ({
  default: (props: { option: unknown }) => {
    received.push({ option: props.option })
    return <div data-testid="echarts-stub" />
  },
}))

function lastOption(): string {
  return JSON.stringify(received.at(-1)?.option ?? null)
}

const WIN_TOKEN_VAR = '--ac-outcome-win'

/** Couleur de remplissage de la bande du premier run, telle que la peint la série personnalisée. */
function firstRunFill(option: unknown): string {
  const series = (option as { series: Array<{ renderItem: (p: unknown, api: unknown) => { children: Array<{ style: { fill: string } }> } }> }).series
  const group = series[0].renderItem({ dataIndex: 0 }, { coord: (v: number[]) => [v[0] * 10, 0] })
  return group.children[0].style.fill
}

describe('suivi de la palette sans remontage', () => {
  const realGetComputedStyle = window.getComputedStyle.bind(window)

  beforeEach(() => {
    received.length = 0
    _resetActivePalette()
    document.documentElement.removeAttribute('style')
    // jsdom ne peuple pas getComputedStyle depuis les propriétés posées par `applyPalette` :
    // on lit les variables de `:root` directement, comme le navigateur les résout.
    vi.spyOn(window, 'getComputedStyle').mockImplementation((el: Element, pseudo?: string | null) => {
      const real = realGetComputedStyle(el, pseudo)
      return new Proxy(real, {
        get(target, prop) {
          if (prop === 'getPropertyValue') {
            return (name: string) =>
              name.startsWith('--ac-')
                ? document.documentElement.style.getPropertyValue(name)
                : target.getPropertyValue(name)
          }
          const value = Reflect.get(target, prop) as unknown
          return typeof value === 'function' ? value.bind(target) : value
        },
      })
    })
    applyPalette(defaultPalette, 'default')
  })

  afterEach(() => {
    vi.restoreAllMocks()
    document.documentElement.removeAttribute('style')
    _resetActivePalette()
  })

  it('les deux palettes de l’essai donnent bien deux couleurs de victoire distinctes', () => {
    expect(defaultPalette['outcome-win']).not.toBe(okabePalette['outcome-win'])
  })

  it('CumulativeFragGapChart : l’option change quand la palette change', async () => {
    render(
      <CumulativeFragGapChart
        points={[
          { cumulative: 3, outcome: 'win' },
          { cumulative: 1, outcome: 'loss' },
        ]}
      />,
    )
    await waitFor(() => expect(received.length).toBeGreaterThan(0))
    const before = lastOption()
    expect(before).toContain(defaultPalette['outcome-win'])
    expect(before).not.toContain(okabePalette['outcome-win'])
    expect(document.documentElement.style.getPropertyValue(WIN_TOKEN_VAR)).toBe(defaultPalette['outcome-win'])

    act(() => applyPalette(okabePalette, 'okabe-ito'))

    await waitFor(() => expect(lastOption()).toContain(okabePalette['outcome-win']))
    expect(lastOption()).not.toContain(defaultPalette['outcome-win'])
  })

  it('OutcomeSequenceTape : l’option est reconstruite et peint la nouvelle couleur quand la palette change', async () => {
    render(
      <OutcomeSequenceTape
        matches={[
          { outcome: 'win', matchId: 'm1' },
          { outcome: 'loss', matchId: 'm2' },
        ]}
        labels={{ win: 'V', loss: 'D', tie: 'N', dnf: 'DNF' }}
      />,
    )
    await waitFor(() => expect(received.length).toBeGreaterThan(0))
    const optionBefore = received.at(-1)?.option
    expect(firstRunFill(optionBefore)).toBe(defaultPalette['outcome-win'])

    act(() => applyPalette(okabePalette, 'okabe-ito'))

    // La couleur d’une série personnalisée est lue par sa fonction de rendu : seule une
    // option RECONSTRUITE fait repeindre ECharts avec la nouvelle palette.
    await waitFor(() => expect(received.at(-1)?.option).not.toBe(optionBefore))
    expect(firstRunFill(received.at(-1)?.option)).toBe(okabePalette['outcome-win'])
  })
})

/**
 * AssistFlowChart — la largeur donnée à la mise en page est celle du CANVAS (le contenu d'un
 * ChartCard a un remplissage de 12 px de chaque côté), pas celle du conteneur mesuré : sinon, à
 * 340 px, les libellés d'un assistant et d'un finisseur à la même hauteur se recouvrent.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { buildAssistFlow, type FlowPair } from './assistFlow.logic'
import { AssistFlowChart, CHART_CARD_PADDING_X } from './AssistFlowChart'
import { flowLayout } from './assistFlowOption'
import { ASSIST_FLOW_TEXT } from './assistFlowStrings'

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => <div data-testid="echarts-stub">{JSON.stringify(option)}</div>,
}))

/** Un ResizeObserver qui annonce tout de suite la largeur du conteneur observé. */
function mockContainerWidth(width: number) {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      cb: (entries: { contentRect: { width: number } }[]) => void
      constructor(cb: (entries: { contentRect: { width: number } }[]) => void) {
        this.cb = cb
      }
      observe() {
        this.cb([{ contentRect: { width } }])
      }
      disconnect() {}
    },
  )
}

afterEach(() => vi.unstubAllGlobals())

const LONG_A = 'UnGamertagTresLong01'
const LONG_B = 'AutreGamertagLong002'
const pair = (a: string, f: string, count: number): FlowPair => ({
  assistantKey: a,
  assistantName: a,
  finisherKey: f,
  finisherName: f,
  count,
  stolen: 0,
})

type Sankey = { left: number; right: number; label: { width: number } }

/** Largeur estimée d'un libellé de 12 px (~7 px par caractère), plafonnée par la troncature. */
const textWidth = (name: string, total: number, cap: number) => Math.min(cap, `${name} · ${total}`.length * 7)

describe('AssistFlowChart — largeur du canvas', () => {
  it('à 340 px, deux gamertags longs à la même hauteur ne se recouvrent pas', async () => {
    mockContainerWidth(340)
    // Un assistant et un finisseur à la même hauteur : un seul ruban entre deux joueurs aux noms longs.
    const flow = buildAssistFlow([pair(LONG_A, LONG_B, 12)])
    render(
      <AssistFlowChart
        flow={flow}
        chart={{ colorOf: () => '#2255cc', fmtShare: String, texts: ASSIST_FLOW_TEXT.fr }}
        headers={{ assistant: 'a', finisher: 'f' }}
        emptyMessage="vide"
      />,
    )
    const text = (await screen.findByTestId('echarts-stub')).textContent ?? ''
    const s = (JSON.parse(text) as { series: Sankey[] }).series[0]
    const canvas = 340 - CHART_CARD_PADDING_X
    // La mise en page a été calculée pour la largeur du canvas, pas celle du conteneur.
    expect(s.left).toBe(flowLayout(canvas).margin)
    expect(s.label.width).toBe(flowLayout(canvas).labelWidth)
    // Libellé de gauche : à droite du nœud de gauche ; libellé de droite : à gauche du nœud de droite.
    const NODE_WIDTH = 14
    const leftEnd = s.left + NODE_WIDTH + textWidth(LONG_A, 12, s.label.width)
    const rightStart = canvas - s.right - NODE_WIDTH - textWidth(LONG_B, 12, s.label.width)
    expect(leftEnd).toBeLessThanOrEqual(rightStart)
  })
})

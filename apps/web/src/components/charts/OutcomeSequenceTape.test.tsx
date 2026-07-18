import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { OutcomeSequenceTape, type OutcomeSequenceLabels } from './OutcomeSequenceTape'
import { matchIndexAtX, type OutcomePoint, type Run } from './outcomeSequence'

// Mock du wrapper ECharts (canvas absent en jsdom) : on capture les props passées
// pour vérifier le câblage `onEvents`/`onChartReady` selon la présence de la prop.
const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: Record<string, unknown>) => {
    captured.push(props)
    return <div data-testid="tape-stub" />
  },
}))

const labels: OutcomeSequenceLabels = { win: 'V', loss: 'D', tie: 'N', dnf: 'DNF' }

function pt(matchId: string, outcome: OutcomePoint['outcome'] = 'win'): OutcomePoint {
  return { outcome, matchId }
}

describe('matchIndexAtX', () => {
  const runs: Run[] = [
    { outcome: 'win', count: 2, matches: [pt('m1'), pt('m2')] },
    { outcome: 'loss', count: 1, matches: [pt('m3', 'loss')] },
  ]

  it('résout le match par index global (offset intra-run)', () => {
    expect(matchIndexAtX(runs, 0.5)?.matchId).toBe('m1')
    expect(matchIndexAtX(runs, 1.9)?.matchId).toBe('m2')
    expect(matchIndexAtX(runs, 2.1)?.matchId).toBe('m3')
  })

  it('borne les X hors limites (négatif → premier, ≥ xMax → dernier)', () => {
    expect(matchIndexAtX(runs, -5)?.matchId).toBe('m1')
    expect(matchIndexAtX(runs, 99)?.matchId).toBe('m3')
    // xMax exact (3) est hors [0, total-1] → borné au dernier.
    expect(matchIndexAtX(runs, 3)?.matchId).toBe('m3')
  })

  it('renvoie null sur runs vides', () => {
    expect(matchIndexAtX([], 0)).toBeNull()
  })

  it('gère un run unique', () => {
    const single: Run[] = [{ outcome: 'win', count: 1, matches: [pt('only')] }]
    expect(matchIndexAtX(single, 0)?.matchId).toBe('only')
    expect(matchIndexAtX(single, 42)?.matchId).toBe('only')
  })
})

describe('OutcomeSequenceTape — non-régression sans prop', () => {
  beforeEach(() => {
    captured.length = 0
  })

  it("ne câble aucun onEvents/onChartReady quand onMatchClick est absent", async () => {
    render(<OutcomeSequenceTape matches={[pt('a'), pt('b', 'loss')]} labels={labels} />)
    await screen.findByTestId('tape-stub')
    const props = captured[captured.length - 1]
    expect(props.onEvents).toBeUndefined()
    expect(props.onChartReady).toBeUndefined()
  })

  it('câble onEvents.click + onChartReady quand onMatchClick est fourni', async () => {
    render(
      <OutcomeSequenceTape matches={[pt('a'), pt('b', 'loss')]} labels={labels} onMatchClick={vi.fn()} />,
    )
    await screen.findByTestId('tape-stub')
    const props = captured[captured.length - 1]
    expect(props.onEvents).toBeDefined()
    expect((props.onEvents as { click?: unknown }).click).toBeTypeOf('function')
    expect(props.onChartReady).toBeTypeOf('function')
  })
})

// GAP #1 (glue non testée) : la résolution du match cliqué dans handleClick — le
// chemin nominal convertFromPixel→matchIndexAtX ET le chemin dégradé (dataIndex)
// quand la conversion pixel est indisponible ou lève. Un bug dans le fallback
// (mauvais dataIndex, mauvais matchId, ou absence de garde Number.isFinite)
// partirait en prod sans être attrapé — matchIndexAtX seul ne le couvre pas.
describe('OutcomeSequenceTape — handleClick (résolution du match)', () => {
  // runs déduits : win×2 [m1,m2], loss×1 [m3].
  const clickMatches = [pt('m1'), pt('m2'), pt('m3', 'loss')]

  function setup() {
    captured.length = 0
    const onMatchClick = vi.fn()
    render(<OutcomeSequenceTape matches={clickMatches} labels={labels} onMatchClick={onMatchClick} />)
    const props = captured[captured.length - 1]
    const click = (props.onEvents as { click: (p: unknown) => void }).click
    const onChartReady = props.onChartReady as (chart: unknown) => void
    return { onMatchClick, click, onChartReady }
  }

  it('résout via convertFromPixel→matchIndexAtX (chemin nominal)', async () => {
    const { onMatchClick, click, onChartReady } = setup()
    await screen.findByTestId('tape-stub')
    // Chart dont convertFromPixel renvoie X=1.9 → index global 1 → m2.
    onChartReady({ convertFromPixel: () => [1.9, 0] })
    click({ dataIndex: 0, event: { offsetX: 50, offsetY: 10 } })
    expect(onMatchClick).toHaveBeenCalledExactlyOnceWith('m2')
  })

  it('retombe sur runs[dataIndex].matches[0] quand convertFromPixel lève', async () => {
    const { onMatchClick, click, onChartReady } = setup()
    await screen.findByTestId('tape-stub')
    onChartReady({
      convertFromPixel: () => {
        throw new Error('boom')
      },
    })
    // dataIndex 1 = run loss → premier match m3 (chemin dégradé).
    click({ dataIndex: 1, event: { offsetX: 5, offsetY: 5 } })
    expect(onMatchClick).toHaveBeenCalledExactlyOnceWith('m3')
  })

  it('retombe sur dataIndex quand convertFromPixel renvoie une valeur non finie', async () => {
    const { onMatchClick, click, onChartReady } = setup()
    await screen.findByTestId('tape-stub')
    onChartReady({ convertFromPixel: () => [Number.NaN, 0] })
    click({ dataIndex: 0, event: { offsetX: 5, offsetY: 5 } })
    expect(onMatchClick).toHaveBeenCalledExactlyOnceWith('m1')
  })

  it('utilise le fallback dataIndex sans chart prêt (offset absent)', async () => {
    const { onMatchClick, click } = setup()
    await screen.findByTestId('tape-stub')
    // onChartReady jamais appelé (chartRef null) + pas d'event → fallback pur.
    click({ dataIndex: 1 })
    expect(onMatchClick).toHaveBeenCalledExactlyOnceWith('m3')
  })

  it("n'émet rien quand ni la conversion ni le dataIndex ne résolvent", async () => {
    const { onMatchClick, click } = setup()
    await screen.findByTestId('tape-stub')
    click({}) // pas de dataIndex, pas d'event
    expect(onMatchClick).not.toHaveBeenCalled()
  })
})

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

/**
 * Tests — ReplayHillSequenceOverlay (la suite des collines jouées, posée sur la carte).
 *
 * CE QU'ILS PROTÈGENT : la pastille n'existe que pour un match à collines, elle montre les collines COMMENCÉES à
 * l'image lue avec la courante marquée, la prochaine n'y figure que si le cycle est établi, et elle se dit dans les
 * deux langues.
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from '../test/testDoc'
import { ReplayHillSequenceOverlay } from './ReplayHillSequenceOverlay'

function zone(zoneRef: number, spans: Array<[number, number]>, active = true) {
  return {
    zoneRef,
    letterRank: zoneRef,
    spans: spans.map(([t0, t1]) => ({ owner: null, active, t0, t1 })),
    gauge: [],
    gaugeRamps: [],
  }
}

/** `count` collines A, B, C, A, ... de 100 images chacune. */
function hillsOf(count: number) {
  const spans: Array<Array<[number, number]>> = [[], [], []]
  for (let i = 0; i < count; i++) spans[i % 3].push([i * 100, i * 100 + 99])
  return testReplayDoc({ frameIntervalMs: 100, zoneStates: spans.map((s, ref) => zone(ref, s)) } as never)
}

describe('ReplayHillSequenceOverlay', () => {
  it('ne rend rien pour un match sans colline', () => {
    const doc = testReplayDoc({ frameIntervalMs: 100, zoneStates: [zone(0, [[0, 500]], false)] } as never)
    const { container } = render(<ReplayHillSequenceOverlay doc={doc} frame={100} locale="fr" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('ne rend rien avant la première colline', () => {
    const doc = testReplayDoc({ frameIntervalMs: 100, zoneStates: [zone(0, [[50, 99]])] } as never)
    const { container } = render(<ReplayHillSequenceOverlay doc={doc} frame={10} locale="fr" />)
    expect(container).toBeEmptyDOMElement()
  })

  it('montre les collines jouées dans l’ordre, la courante marquée, sans prochaine tant que le cycle n’est pas établi', () => {
    render(<ReplayHillSequenceOverlay doc={hillsOf(7)} frame={120} locale="fr" />)
    const pastille = screen.getByRole('status')
    expect(pastille).toHaveTextContent('Collines')
    expect(screen.getByText('A')).toBeInTheDocument()
    expect(screen.getByText('B')).toHaveAttribute('aria-current', 'step')
    expect(screen.queryByText(/Prochaine/)).toBeNull()
    expect(pastille).toHaveAccessibleName('Collines jouées : A, B. Colline en cours : B')
  })

  it('dit la prochaine colline quand le cycle est répété en entier', () => {
    render(<ReplayHillSequenceOverlay doc={hillsOf(7)} frame={550} locale="fr" />)
    expect(screen.getByText('· Prochaine A')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveAccessibleName(
      'Collines jouées : A, B, C, A, B, C. Colline en cours : C. Prochaine : A',
    )
  })

  it('se dit en anglais', () => {
    render(<ReplayHillSequenceOverlay doc={hillsOf(7)} frame={550} locale="en" />)
    expect(screen.getByText('· Next A')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Hills')
  })

  it('ne garde que les dernières collines quand la suite est longue', () => {
    render(<ReplayHillSequenceOverlay doc={hillsOf(31)} frame={2_950} locale="fr" />)
    expect(screen.getAllByText(/^[ABC]$/)).toHaveLength(8)
    expect(screen.getByText('…')).toBeInTheDocument()
  })
})

/**
 * TimeseriesIntensityProfile.test.tsx — « Intensité » solo (onglet Progression).
 *
 * Panneau unique réutilisant le builder P1 en N=1. Vérifie : rendu quand au
 * moins une manche a des frags, état vide (message dans le bloc titré) quand
 * aucune manche exploitable ou liste vide. echarts-for-react mocké (canvas jsdom).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { IntensityMatchRow } from '@/lib/api/types'
import { TimeseriesIntensityProfile } from './TimeseriesSquadAdapted'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

function row(phases: number[] | null, id = 'm1', label = 'Aquarius — 30/04'): IntensityMatchRow {
  return { match_id: id, label, phases } as IntensityMatchRow
}

/** N manches concentrées sur la phase 0 (exploitables). */
function exploitableRows(n: number): IntensityMatchRow[] {
  return Array.from({ length: n }, (_, i) => {
    const p = new Array<number>(10).fill(0)
    p[0] = 1
    return row(p, `m${i}`)
  })
}

const LABELS = { medianLabel: 'Médiane', envelopeLabel: 'Enveloppe P25–P75', refLabel: '10 %' }

afterEach(() => vi.clearAllMocks())

describe('TimeseriesIntensityProfile', () => {
  it('rend le chart quand au moins une manche a des frags', async () => {
    render(
      <TimeseriesIntensityProfile rows={exploitableRows(5)} title="Intensité" emptyMessage="vide" {...LABELS} />,
    )
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
  })

  it('liste vide → état vide (message dans le bloc titré)', () => {
    render(<TimeseriesIntensityProfile rows={[]} title="Intensité" emptyMessage="Aucune donnée" {...LABELS} />)
    expect(screen.getByTestId('chart-card-empty')).toBeInTheDocument()
    expect(screen.getByText('Aucune donnée')).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })

  it('manches sans frag (Σ = 0) → état vide (builder sans série)', () => {
    const zero = new Array<number>(10).fill(0)
    render(
      <TimeseriesIntensityProfile
        rows={[row(zero, 'm1'), row(null, 'm2')]}
        title="Intensité"
        emptyMessage="Aucune donnée"
        {...LABELS}
      />,
    )
    expect(screen.getByTestId('chart-card-empty')).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })
})

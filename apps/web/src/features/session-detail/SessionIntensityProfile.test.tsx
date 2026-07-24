/**
 * SessionIntensityProfile.test.tsx — « Intensité » solo (page session).
 *
 * Panneau unique réutilisant le builder P1 en N=1 sur `intensity_rows` du payload
 * session. Vérifie : rendu quand au moins une manche a des frags, état vide (message
 * dans le bloc titré) quand aucune manche exploitable / liste vide. echarts-for-react
 * mocké (canvas jsdom).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import type { IntensityMatchRow } from '@/lib/api/types'

import { SessionIntensityProfile } from './SessionIntensityProfile'

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

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

afterEach(() => vi.clearAllMocks())

describe('SessionIntensityProfile', () => {
  it('rend le chart + le sous-titre quand au moins une manche a des frags', async () => {
    render(<SessionIntensityProfile title="Intensité" rows={exploitableRows(5)} />)
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    expect(screen.getByText('Répartition des frags par phase de match')).toBeInTheDocument()
  })

  it('liste vide → état vide (message dans le bloc titré, pas de chart)', () => {
    render(<SessionIntensityProfile title="Intensité" rows={[]} />)
    expect(screen.getByTestId('chart-card-empty')).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })

  it('manches sans frag (Σ = 0) → état vide', () => {
    const zero = new Array<number>(10).fill(0)
    render(<SessionIntensityProfile title="Intensité" rows={[row(zero, 'm1'), row(null, 'm2')]} />)
    expect(screen.getByTestId('chart-card-empty')).toBeInTheDocument()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })
})

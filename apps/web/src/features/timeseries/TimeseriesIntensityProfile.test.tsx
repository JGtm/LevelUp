/**
 * TimeseriesIntensityProfile.test.tsx — « Intensité » solo (onglet Progression).
 *
 * Panneau unique réutilisant le builder P1 en N=1. Vérifie : rendu quand au
 * moins une manche a des frags, état vide (message dans le bloc titré) quand
 * aucune manche exploitable ou liste vide, et — depuis le 2026-09-19 — les deux
 * COURBES DE RÉFÉRENCE (équipe alliée, lobby entier) avec leur légende.
 * echarts-for-react mocké (canvas jsdom).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { IntensityMatchRow } from '@/lib/api/types'
import { TimeseriesIntensityProfile } from './TimeseriesSquadAdapted'

const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: Record<string, unknown>) => {
    captured.push(props)
    return <div data-testid="echarts-mock" />
  },
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

const LABELS = {
  medianLabel: 'Médiane',
  envelopeLabel: 'Enveloppe P25–P75',
  refLabel: '10 %',
  playerLabel: 'Joueur',
  teamLabel: 'Équipe',
  lobbyLabel: 'Lobby',
}

beforeEach(() => {
  captured.length = 0
})
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
    // Assertion PORTÉE SUR LE BLOC : depuis l'alignement de l'état vide des cartes de
    // graphe sur `EmptyStateNotice` (2026-09-22), le bloc porte un titre par défaut
    // (« Aucune donnée ») en plus de la description — ici la fixture emploie le même
    // libellé, et un `getByText` global y trouverait deux nœuds.
    expect(screen.getByTestId('chart-card-empty')).toHaveTextContent('Aucune donnée')
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

  // 2026-09-19 (item 1.G) : les deux courbes de référence et la légende à trois entrées.
  it('monte les courbes ÉQUIPE et LOBBY servies, et les nomme dans la légende', async () => {
    render(
      <TimeseriesIntensityProfile
        rows={exploitableRows(5)}
        teamRows={exploitableRows(5)}
        lobbyRows={exploitableRows(5)}
        title="Intensité"
        emptyMessage="vide"
        {...LABELS}
      />,
    )
    await screen.findByTestId('echarts-mock')
    const option = captured[captured.length - 1].option as {
      legend: { data: Array<{ name: string }> }
      series: Array<{ name?: string }>
    }
    expect(option.legend.data.map((e) => e.name)).toEqual(['Joueur', 'Équipe', 'Lobby'])
    expect(option.series.some((s) => s.name === 'Équipe')).toBe(true)
    expect(option.series.some((s) => s.name === 'Lobby')).toBe(true)
  })

  it('une courbe de référence ABSENTE du contrat n’est pas montée (jamais une courbe plate)', async () => {
    render(
      <TimeseriesIntensityProfile
        rows={exploitableRows(5)}
        title="Intensité"
        emptyMessage="vide"
        {...LABELS}
      />,
    )
    await screen.findByTestId('echarts-mock')
    const option = captured[captured.length - 1].option as {
      legend: { data: Array<{ name: string }> }
    }
    expect(option.legend.data.map((e) => e.name)).toEqual(['Joueur'])
  })
})

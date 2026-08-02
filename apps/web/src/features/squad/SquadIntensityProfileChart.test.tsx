/**
 * SquadIntensityProfileChart.test.tsx — « Intensité » (onglet Dynamique).
 *
 * Vérifie : rendu du chart quand au moins un joueur a des manches exploitables,
 * état vide sinon, exclusion de la ligne agrégée `all` et respect de l'ordre des
 * joueurs (playerOrder). Le builder ECharts est mocké pour capturer les panneaux
 * construits côté composant.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { SquadIntensityProfile } from '@/lib/api/types'
import { getSquadText } from './i18n'
import { SquadIntensityProfileChart } from './SquadIntensityProfileChart'
import type { IntensityPanelInput } from './charts/squadIntensityProfileChart'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const captured: {
  panels?: IntensityPanelInput[]
  teamOverlay?: { label: string; rows: Array<{ phases: number[] | null }> }
} = {}
vi.mock('./charts/squadIntensityProfileChart', () => ({
  buildSquadIntensityProfileOption: (opts: {
    panels: IntensityPanelInput[]
    teamOverlay?: { label: string; rows: Array<{ phases: number[] | null }> }
  }) => {
    captured.panels = opts.panels
    captured.teamOverlay = opts.teamOverlay
    return { backgroundColor: 'transparent' }
  },
  intensityAxisLabels: () => ({ start: 'Début', mid: 'Milieu', end: 'Fin', rangeSuffix: 'du match' }),
}))

const T = getSquadText('fr')

/** N manches concentrées sur la phase 0 (exploitables). */
function rows(n: number): Array<{ phases: number[] }> {
  return Array.from({ length: n }, () => {
    const p = new Array<number>(10).fill(0)
    p[0] = 1
    return { phases: p }
  })
}

const COLORS = { Me: '#aaa', F1: '#bbb' }

function profileWith(rowsByKey: Record<string, Array<{ phases: number[] | null }>>): SquadIntensityProfile {
  return {
    options: Object.keys(rowsByKey).map((key) => ({ key, label: key })),
    rows: rowsByKey as SquadIntensityProfile['rows'],
  }
}

function renderChart(profile: SquadIntensityProfile, playerOrder?: string[]) {
  return render(
    <SquadIntensityProfileChart
      title={T.intensity.title}
      subtitle={T.intensity.subtitle}
      tooltip={T.intensity.tooltip}
      medianLabel={T.intensity.medianLabel}
      envelopeLabel={T.intensity.envelopeLabel}
      refLabel={T.intensity.refLabel}
      teamLabel={T.intensity.teamLabel}
      emptyMessage={T.empty.noBlockData}
      profile={profile}
      colorByPlayer={COLORS}
      playerOrder={playerOrder}
    />,
  )
}

afterEach(() => {
  captured.panels = undefined
  captured.teamOverlay = undefined
  vi.clearAllMocks()
})

describe('SquadIntensityProfileChart', () => {
  it('rend le chart + le sous-titre quand un joueur a des manches exploitables', async () => {
    renderChart(profileWith({ all: rows(5), Me: rows(5), F1: rows(5) }), ['Me', 'F1'])
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    expect(screen.getByText(T.intensity.subtitle)).toBeInTheDocument()
  })

  it('exclut la ligne agrégée `all` et respecte playerOrder', async () => {
    renderChart(profileWith({ all: rows(5), Me: rows(5), F1: rows(5) }), ['Me', 'F1'])
    await screen.findByTestId('echarts-mock')
    expect(captured.panels?.map((p) => p.key)).toEqual(['Me', 'F1'])
  })

  it('état vide quand aucun joueur n a de manche exploitable', () => {
    const empty = new Array<number>(10).fill(0)
    renderChart(profileWith({ Me: [{ phases: empty }], F1: [{ phases: null }] }), ['Me', 'F1'])
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
    expect(screen.getByText(T.empty.noBlockData)).toBeInTheDocument()
  })

  it('couleur de panneau = colorByPlayer par gamertag', async () => {
    renderChart(profileWith({ all: rows(5), Me: rows(5) }), ['Me'])
    await screen.findByTestId('echarts-mock')
    expect(captured.panels?.[0].color).toBe('#aaa')
  })
})

describe('SquadIntensityProfileChart — courbe agrégée d équipe', () => {
  it('3 joueurs : la ligne `all` du payload est passée en courbe d équipe', async () => {
    renderChart(
      profileWith({ all: rows(6), Me: rows(5), F1: rows(5), F2: rows(5) }),
      ['Me', 'F1', 'F2'],
    )
    await screen.findByTestId('echarts-mock')
    expect(captured.panels).toHaveLength(3)
    expect(captured.teamOverlay?.label).toBe(T.intensity.teamLabel)
    expect(captured.teamOverlay?.rows).toHaveLength(6)
  })

  it('2 joueurs : aucune courbe d équipe (comparaison directe suffisante)', async () => {
    renderChart(profileWith({ all: rows(6), Me: rows(5), F1: rows(5) }), ['Me', 'F1'])
    await screen.findByTestId('echarts-mock')
    expect(captured.panels).toHaveLength(2)
    expect(captured.teamOverlay).toBeUndefined()
  })

  it('3 joueurs mais ligne `all` absente : dégradation silencieuse', async () => {
    renderChart(profileWith({ Me: rows(5), F1: rows(5), F2: rows(5) }), ['Me', 'F1', 'F2'])
    await screen.findByTestId('echarts-mock')
    expect(captured.panels).toHaveLength(3)
    expect(captured.teamOverlay).toBeUndefined()
  })

  it('3 joueurs mais ligne `all` sans frag : pas de courbe d équipe plate', async () => {
    const empty = new Array<number>(10).fill(0)
    renderChart(
      profileWith({ all: [{ phases: empty }], Me: rows(5), F1: rows(5), F2: rows(5) }),
      ['Me', 'F1', 'F2'],
    )
    await screen.findByTestId('echarts-mock')
    expect(captured.teamOverlay).toBeUndefined()
  })
})

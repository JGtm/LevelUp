/**
 * SquadIntensityProfileChart.test.tsx — « Intensité » (onglet Dynamique).
 *
 * Vérifie : rendu du chart quand au moins un joueur a des manches exploitables,
 * état vide sinon, exclusion des lignes agrégées `team` / `lobby`, respect de
 * l'ordre des joueurs (playerOrder) et montage des deux courbes de référence
 * (lobby dès 1 joueur, équipe à partir de 3). Le builder ECharts est mocké pour capturer les panneaux
 * construits côté composant.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { intensityTooltipText } from '@/components/charts/intensityTooltipText'
import type { SquadIntensityProfile } from '@/lib/api/types'
import { getSquadText } from './i18n'
import { SquadIntensityProfileChart } from './SquadIntensityProfileChart'
import type { IntensityPanelInput } from './charts/squadIntensityProfileChart'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

type CapturedOverlay = { key: string; label: string; rows: Array<{ phases: number[] | null }> }
const captured: {
  panels?: IntensityPanelInput[]
  overlays?: CapturedOverlay[]
} = {}
vi.mock('./charts/squadIntensityProfileChart', () => ({
  buildSquadIntensityProfileOption: (opts: {
    panels: IntensityPanelInput[]
    overlays?: CapturedOverlay[]
  }) => {
    captured.panels = opts.panels
    captured.overlays = opts.overlays
    return { backgroundColor: 'transparent' }
  },
  intensityAxisLabels: () => ({ start: 'Début', mid: 'Milieu', end: 'Fin', rangeSuffix: 'du match' }),
  isIntensityOverlayKey: (key: string) => key === 'team' || key === 'lobby',
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
      tooltip={intensityTooltipText('fr', { withTeam: true })}
      medianLabel={T.intensity.medianLabel}
      envelopeLabel={T.intensity.envelopeLabel}
      refLabel={T.intensity.refLabel}
      teamLabel={T.intensity.teamLabel}
      lobbyLabel={T.intensity.lobbyLabel}
      emptyMessage={T.empty.noBlockData}
      profile={profile}
      colorByPlayer={COLORS}
      playerOrder={playerOrder}
    />,
  )
}

afterEach(() => {
  captured.panels = undefined
  captured.overlays = undefined
  vi.clearAllMocks()
})

describe('SquadIntensityProfileChart', () => {
  it('rend le chart + le sous-titre quand un joueur a des manches exploitables', async () => {
    renderChart(profileWith({ lobby: rows(5), Me: rows(5), F1: rows(5) }), ['Me', 'F1'])
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    expect(screen.getByText(T.intensity.subtitle)).toBeInTheDocument()
  })

  it('exclut la ligne agrégée `lobby` et respecte playerOrder', async () => {
    renderChart(profileWith({ lobby: rows(5), Me: rows(5), F1: rows(5) }), ['Me', 'F1'])
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
    renderChart(profileWith({ lobby: rows(5), Me: rows(5) }), ['Me'])
    await screen.findByTestId('echarts-mock')
    expect(captured.panels?.[0].color).toBe('#aaa')
  })
})

describe('SquadIntensityProfileChart — courbes de référence équipe / lobby', () => {
  const keys = () => captured.overlays?.map((o) => o.key)

  it('3 joueurs : équipe (`team`) ET lobby (`lobby`) sont montés, avec leurs libellés', async () => {
    renderChart(
      profileWith({ team: rows(6), lobby: rows(7), Me: rows(5), F1: rows(5), F2: rows(5) }),
      ['Me', 'F1', 'F2'],
    )
    await screen.findByTestId('echarts-mock')
    expect(captured.panels).toHaveLength(3)
    expect(keys()).toEqual(['team', 'lobby'])
    expect(captured.overlays?.[0].label).toBe(T.intensity.teamLabel)
    expect(captured.overlays?.[0].rows).toHaveLength(6)
    expect(captured.overlays?.[1].label).toBe(T.intensity.lobbyLabel)
    expect(captured.overlays?.[1].rows).toHaveLength(7)
  })

  it('1 joueur : le lobby est monté, pas l équipe (seuil 3 joueurs)', async () => {
    renderChart(profileWith({ team: rows(6), lobby: rows(7), Me: rows(5) }), ['Me'])
    await screen.findByTestId('echarts-mock')
    expect(captured.panels).toHaveLength(1)
    expect(keys()).toEqual(['lobby'])
  })

  it('2 joueurs : lobby seul (comparaison directe des joueurs, pas de courbe équipe)', async () => {
    renderChart(profileWith({ team: rows(6), lobby: rows(7), Me: rows(5), F1: rows(5) }), ['Me', 'F1'])
    await screen.findByTestId('echarts-mock')
    expect(captured.panels).toHaveLength(2)
    expect(keys()).toEqual(['lobby'])
  })

  it('lignes `team` / `lobby` absentes du payload : dégradation silencieuse, aucun overlay', async () => {
    renderChart(profileWith({ Me: rows(5), F1: rows(5), F2: rows(5) }), ['Me', 'F1', 'F2'])
    await screen.findByTestId('echarts-mock')
    expect(captured.panels).toHaveLength(3)
    expect(keys()).toEqual([])
  })

  it('ligne agrégée sans frag : pas de courbe plate, l autre reste montée', async () => {
    const empty = new Array<number>(10).fill(0)
    renderChart(
      profileWith({ team: [{ phases: empty }], lobby: rows(7), Me: rows(5), F1: rows(5), F2: rows(5) }),
      ['Me', 'F1', 'F2'],
    )
    await screen.findByTestId('echarts-mock')
    expect(keys()).toEqual(['lobby'])
  })

  it('les clés agrégées ne deviennent jamais des panneaux (ordre par défaut sans playerOrder)', async () => {
    renderChart(profileWith({ team: rows(6), lobby: rows(7), Me: rows(5), F1: rows(5) }))
    await screen.findByTestId('echarts-mock')
    expect(captured.panels?.map((p) => p.key)).toEqual(['Me', 'F1'])
  })
})

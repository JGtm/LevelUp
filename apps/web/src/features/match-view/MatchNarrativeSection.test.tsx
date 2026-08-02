import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

import { MatchNarrativeSection, type MatchNarrativeLabels } from './MatchNarrativeSection'
import type {
  MatchViewCadence,
  MatchViewHeader,
  MatchViewImpactRole,
  MatchViewRadarSeries,
} from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `var(${token})`,
  resolveToken: (token: string) => `var(${token})`,
}))

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option).slice(0, 60)}</div>
  ),
}))

const baseLabels: MatchNarrativeLabels = {
  sectionTitle: 'Narratif',
  cadenceTitle: 'Cadence',
  impactSectionTitle: 'Rôles',
  radarSectionTitle: 'Profil',
  resolveLabelKey: (key) => `[${key}]`,
  resolveAxisLabel: (axis) => axis.toUpperCase(),
}

const headerWithoutBadge: MatchViewHeader = {
  match_id: 'm1',
  start_time: undefined,
  start_time_label: '',
  outcome_code: 2,
  outcome_label: 'Win',
  outcome_color: '#22c55e',
  score_label: '',
  dominance_flag: false,
  had_bot_teammate: false,
  map_ui: 'Aquarius',
  map_id: undefined,
  mode_ui: 'Slayer',
  playlist_label: 'Ranked',
  performance_display: '85',
  performance_color: undefined,
  is_excluded: false,
  is_ranked: false,
  is_favorite: false,
  replay_available: false,
}

describe('MatchNarrativeSection', () => {
  it('rend null si aucun contenu narratif', () => {
    const { container } = render(
      <MatchNarrativeSection header={headerWithoutBadge} labels={baseLabels} />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('rend la pill DominanceBadge quand dominance_badge est présent', () => {
    const header: MatchViewHeader = {
      ...headerWithoutBadge,
      dominance_flag: true,
      dominance_badge: {
        flag: 1,
        label_key: 'narrative.dominance.domination',
        color_token: 'narrative-dominant',
      },
    }
    render(<MatchNarrativeSection header={header} labels={baseLabels} />)
    const badge = screen.getByTestId('match-narrative-dominance-badge')
    expect(badge.textContent).toContain('[narrative.dominance.domination]')
  })

  it('rend la liste des rôles d\'impact avec gamertag résolu', () => {
    const roles: MatchViewImpactRole[] = [
      {
        xuid: 'x_p1',
        role_key: 'first_blood',
        label_key: 'narrative.role.first_blood',
        color_token: 'narrative-dominant',
      },
      {
        xuid: 'x_p2',
        role_key: 'top_killer',
        label_key: 'narrative.role.top_killer',
        color_token: 'narrative-dominant',
      },
    ]
    render(
      <MatchNarrativeSection
        header={headerWithoutBadge}
        impactRoles={roles}
        labels={baseLabels}
        gamertagByXUID={{ x_p1: 'PlayerOne', x_p2: 'PlayerTwo' }}
      />,
    )
    const list = screen.getByTestId('match-narrative-impact-roles')
    expect(list.textContent).toContain('PlayerOne')
    expect(list.textContent).toContain('[narrative.role.first_blood]')
    expect(list.textContent).toContain('PlayerTwo')
  })

  it('rend la cadence via le wrapper BarStackedChart', async () => {
    const cadence: MatchViewCadence = {
      key: 'match_view.combat.cadence',
      datapoints: [
        { category: 'phase_00', components: { x_p1: 1, x_p2: 0 } },
        { category: 'phase_01', components: { x_p1: 2, x_p2: 1 } },
      ],
    }
    render(
      <MatchNarrativeSection
        header={headerWithoutBadge}
        cadence={cadence}
        labels={baseLabels}
      />,
    )
    // Le wrapper BarStackedChart utilise Suspense + lazy import — attendre.
    await waitFor(() => {
      expect(screen.getAllByTestId('echarts-stub').length).toBeGreaterThan(0)
    })
  })

  it('rend le radar via le wrapper RadarChart', async () => {
    const radar: MatchViewRadarSeries[] = [
      {
        xuid: 'x_p1',
        gamertag: 'PlayerOne',
        axes: [
          { Axis: 'combat', Value: 80, Raw: 200 },
          { Axis: 'survival', Value: 50, Raw: 25 },
          { Axis: 'support', Value: 30, Raw: 24 },
          { Axis: 'score', Value: 60, Raw: 48 },
          { Axis: 'objective', Value: 0, Raw: 0 },
          { Axis: 'impact', Value: 75, Raw: 75 },
        ],
        mode_family: 'slayer',
      },
    ]
    render(
      <MatchNarrativeSection header={headerWithoutBadge} radar={radar} labels={baseLabels} />,
    )
    await waitFor(() => {
      expect(screen.getAllByTestId('echarts-stub').length).toBeGreaterThan(0)
    })
  })

  it('combine plusieurs sections quand toutes ont du contenu', () => {
    const header: MatchViewHeader = {
      ...headerWithoutBadge,
      dominance_badge: {
        flag: 3,
        label_key: 'narrative.dominance.remontada',
        color_token: 'narrative-remontada',
      },
    }
    render(
      <MatchNarrativeSection
        header={header}
        impactRoles={[
          { xuid: 'x', role_key: 'top_killer', label_key: 'narrative.role.top_killer', color_token: 'narrative-dominant' },
        ]}
        cadence={{
          key: 'k',
          datapoints: [{ category: 'phase_00', components: { x: 1 } }],
        }}
        labels={baseLabels}
      />,
    )
    expect(screen.getByTestId('match-narrative-section')).toBeInTheDocument()
    expect(screen.getByTestId('match-narrative-dominance-badge')).toBeInTheDocument()
    expect(screen.getByTestId('match-narrative-impact-roles')).toBeInTheDocument()
  })
})

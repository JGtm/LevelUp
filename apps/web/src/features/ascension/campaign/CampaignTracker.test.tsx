/**
 * CampaignTracker — un départ sans échantillon (`snapshot_sample` = 0) n'est pas rendu comme une valeur :
 * un tiret au départ et à l'écart, aucune progression chiffrée, aucun texte ni action à la place (garde-rail
 * « aucun inconnu à l'écran »).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import type { ImprovementCampaign } from '@/lib/playerProfile'
import { campaignDelta, isStartMeasured } from './campaignTrackerLogic'

const mockShellState = { locale: 'fr' as 'fr' | 'en' }

vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: <T,>(selector: (s: typeof mockShellState) => T) => selector(mockShellState),
}))

const idle = { mutate: vi.fn(), isPending: false }
vi.mock('@/features/ascension/profile/queries', () => ({
  useCampaignMutations: () => ({ pause: idle, resume: idle, close: idle, abandon: idle }),
}))

import { CampaignTracker } from './CampaignTracker'

function campaign(over: Partial<ImprovementCampaign>): ImprovementCampaign {
  return {
    id: 'c1',
    user_id: 'u1',
    title_slug: 'halo_infinite',
    axis: 'objective',
    axis_kind: 'radar',
    status: 'active',
    playlist_group: 'all',
    started_at: '2026-10-01T10:00:00Z',
    snapshot_value: 0,
    snapshot_sample: 0,
    matches_since_start: 25,
    current_value_lowess: 55,
    current_value_raw: 55,
    progression_confirmed: false,
    auto_closure_suggested: false,
    ...over,
  } as ImprovementCampaign
}

describe('campaignTrackerLogic', () => {
  it('un départ sans échantillon est non mesuré et sans écart', () => {
    expect(isStartMeasured({ snapshot_sample: 0 })).toBe(false)
    expect(isStartMeasured({ snapshot_sample: 12 })).toBe(true)
    expect(campaignDelta({ snapshot_sample: 0, snapshot_value: 0, current_value_lowess: 55 })).toBeUndefined()
    expect(campaignDelta({ snapshot_sample: 12, snapshot_value: 40, current_value_lowess: 55 })).toBe(15)
    expect(campaignDelta({ snapshot_sample: 12, snapshot_value: 40, current_value_lowess: undefined })).toBeUndefined()
  })
})

describe('CampaignTracker — départ sans échantillon', () => {
  beforeEach(() => {
    cleanup()
    mockShellState.locale = 'fr'
  })

  it("affiche un tiret au départ et à l'écart, jamais « 0.00 », sans texte d'explication", () => {
    render(<CampaignTracker playerSlug="demo" campaign={campaign({})} />)
    expect(screen.queryByText('0.00')).not.toBeInTheDocument()
    expect(screen.getAllByText('—')).toHaveLength(2)
    expect(screen.getByText('55.00')).toBeInTheDocument()
    expect(screen.queryByText('+55.00')).not.toBeInTheDocument()
  })

  it('un départ mesuré affiche sa valeur et l\'écart chiffré, sans note', () => {
    render(
      <CampaignTracker
        playerSlug="demo"
        campaign={campaign({ snapshot_value: 40, snapshot_sample: 12 })}
      />,
    )
    expect(screen.getByText('40.00')).toBeInTheDocument()
    expect(screen.getByText('+15.00')).toBeInTheDocument()
  })
})

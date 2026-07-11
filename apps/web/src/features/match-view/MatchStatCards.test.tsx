import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MatchSummaryCardsSection } from './MatchStatCards'
import type { MatchSummaryKpis, MatchExpectedStats } from '@/lib/api/types'

// Capabilities pilotables : Halo 5 = pas de damage_taken ni de team_mmr.
const caps = { damageTaken: true, teamMmr: true }
vi.mock('@/lib/damage/effectiveHp', () => ({
  useProvidesDamageTaken: () => caps.damageTaken,
  useProvidesTeamMmr: () => caps.teamMmr,
}))

// Locale FR fixe (le sélecteur reçoit un state minimal).
vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: (sel: (s: { locale: string }) => unknown) => sel({ locale: 'fr' }),
}))

const kpis: MatchSummaryKpis = {
  kills: 15,
  deaths: 10,
  assists: 5,
  team_mmr: 1500,
  enemy_mmr: 1450,
  delta_mmr: 50,
  average_life: '0:42',
}

function renderSection(overrides?: {
  expected?: Partial<MatchExpectedStats>
  damageTaken?: boolean
  teamMmr?: boolean
}) {
  caps.damageTaken = overrides?.damageTaken ?? true
  caps.teamMmr = overrides?.teamMmr ?? true
  const expectedStats: MatchExpectedStats = {
    expected_kills: 12,
    expected_deaths: 11,
    expected_assists: 6,
    expected_win_prob: 0.62,
    has_hist_avg: false,
    locally_estimated: false,
    ...overrides?.expected,
  }
  return render(
    <MatchSummaryCardsSection
      kpis={kpis}
      expectedStats={expectedStats}
      offensiveConversion={1.2}
      defensiveResistance={1.1}
      damagePerKill={120}
      damagePerDeath={95}
    />,
  )
}

describe('MatchSummaryCardsSection — masquage des cards par capability (résidus H5)', () => {
  beforeEach(() => {
    caps.damageTaken = true
    caps.teamMmr = true
  })

  it('Infinite (capabilities + winProb) : cards Résistance ET Résultat attendu présentes', () => {
    renderSection()
    expect(screen.getByText('Résistance')).toBeInTheDocument()
    expect(screen.getByText('Résultat attendu')).toBeInTheDocument()
    expect(screen.getByText('MMR')).toBeInTheDocument()
  })

  it('Halo 5 (pas de damage_taken) : card Résistance ABSENTE (C1/DEC-2)', () => {
    renderSection({ damageTaken: false })
    expect(screen.queryByText('Résistance')).not.toBeInTheDocument()
    // Le Rendement (offensif, sans damage_taken) reste affiché.
    expect(screen.getByText('Rendement')).toBeInTheDocument()
  })

  it('Halo 5 (pas de team_mmr) : card MMR ABSENTE (non-régression)', () => {
    renderSection({ teamMmr: false })
    expect(screen.queryByText('MMR')).not.toBeInTheDocument()
  })

  it('winProb absent : card Résultat attendu ABSENTE (C2/DEC-3)', () => {
    renderSection({ expected: { expected_win_prob: undefined } })
    expect(screen.queryByText('Résultat attendu')).not.toBeInTheDocument()
  })

  it('winProb présent : card Résultat attendu affiche le pourcentage', () => {
    renderSection({ expected: { expected_win_prob: 0.62 } })
    expect(screen.getByText('Résultat attendu')).toBeInTheDocument()
    expect(screen.getByText('62 %')).toBeInTheDocument()
  })
})

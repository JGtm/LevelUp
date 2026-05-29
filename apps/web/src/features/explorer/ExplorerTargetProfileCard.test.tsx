/**
 * Tests ExplorerTargetProfileCard — couverture des 4 états :
 *  - tout dispo (identity + career + sample + privacy=none)
 *  - profil privé (privacy=full → bannière, career absent)
 *  - no-tokens (auth_available=false → hint affiché, career masquée)
 *  - sample vide (sample_size=0 → section sample masquée)
 */
import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerTargetProfile } from '@/lib/api/types'

import { ExplorerTargetProfileCard } from './ExplorerTargetProfileCard'

const SAMPLE_FULL: ExplorerTargetProfile['sample_stats'] = {
  sample_size: 12,
  kills: 50,
  deaths: 20,
  assists: 15,
  wins: 7,
  losses: 4,
  draws: 1,
  shots_fired: 800,
  shots_hit: 400,
  damage_dealt: 9000,
  damage_taken: 6000,
  headshot_kills: 18,
  melee_kills: 5,
  power_weapon_kills: 10,
  grenade_kills: 7,
  total_medals: 42,
  unique_medals: 8,
  kda: 2.75,
  kdr: 2.5,
  win_rate: 0.583,
  accuracy: 0.5,
  headshot_rate: 0.36,
  offensive_conversion: 1.2,
  defensive_resistance: 1.3,
  kills_per_min: 1.1,
  deaths_per_min: 0.45,
  assists_per_min: 0.33,
  avg_personal_score: 1850,
  perfect_kills: 6,
}

const IDENTITY_FULL: ExplorerTargetProfile['identity'] = {
  banner_image_url: '/api/v1/assets/spartan/banner/x.png',
  emblem_image_url: '/api/v1/assets/spartan/emblem/x.png',
  spartan_id: 'ABCD',
  career_rank: {
    rank_number: 76,
    rank_title: 'Hero',
    current_xp: 47820,
    xp_for_next_rank: 50000,
    progress_pct: 0.95,
    is_max_rank: false,
  },
  highest_csr: null,
  highest_lusr: null,
}

const CAREER_FULL: ExplorerTargetProfile['career_stats'] = {
  xuid: 'xuid-target',
  gamertag: 'TargetPlayer',
  title_slug: 'halo_infinite',
  is_local: false,
  matches: 1247,
  win_rate: 0.583,
  kda: 1.34,
  kdr: 1.21,
  kills_per_game: 12,
  deaths_per_game: 10,
  assists_per_game: 5,
  accuracy: 0.472,
  damage_per_game: 1843,
  damage_taken_per_game: 1500,
  perfect_kills_per_game: 0.3,
  max_killing_spree: 14,
  avg_life_secs: 90,
  headshot_kills_per_game: 4.1,
  perf_ath: 0,
  lusr_ath: 0,
  career_rank: 76,
}

describe('ExplorerTargetProfileCard', () => {
  it('rend les 4 sections quand tout est dispo', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: CAREER_FULL,
      sample_stats: SAMPLE_FULL,
      privacy_warning: { level: 'none', message: '' },
      auth_available: true,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="TargetPlayer" />)

    expect(screen.getByTestId('explorer-target-profile-card')).toBeInTheDocument()
    // Identité : gamertag affiché
    expect(screen.getByText('TargetPlayer')).toBeInTheDocument()
    // Career : présent (label Matches du manifest fr)
    expect(screen.getByText(/Carrière complète/i)).toBeInTheDocument()
    // Sample : section présente avec le count
    expect(screen.getByText(/12 matchs joués ensemble/i)).toBeInTheDocument()
    // Privacy : level=none → pas de bannière (return null dans PrivacyBanner)
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    // No-auth hint : absent
    expect(screen.queryByTestId('explorer-target-no-auth-hint')).not.toBeInTheDocument()
  })

  it('affiche le PrivacyBanner pour un profil privé', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: null,
      sample_stats: SAMPLE_FULL,
      privacy_warning: { level: 'full', message: 'Profil privé.' },
      auth_available: true,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="PrivatePlayer" />)

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText('Profil privé.')).toBeInTheDocument()
    // Career absent → pas de section
    expect(screen.queryByText(/Carrière complète/i)).not.toBeInTheDocument()
    // Identité reste affichée
    expect(screen.getByText('PrivatePlayer')).toBeInTheDocument()
  })

  it('affiche le hint no-auth quand auth_available=false et career absente', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: null,
      sample_stats: SAMPLE_FULL,
      privacy_warning: null,
      auth_available: false,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="NoAuthPlayer" />)

    expect(screen.getByTestId('explorer-target-no-auth-hint')).toBeInTheDocument()
    // La section "Carrière complète" (CardHeader) est absente — on cible le subtitle
    // unique pour distinguer du texte de hint qui mentionne aussi "carrière".
    expect(screen.queryByText(/via API Halo/i)).not.toBeInTheDocument()
    // Sample reste affiché (calcul local indépendant des tokens)
    expect(screen.getByText(/12 matchs joués ensemble/i)).toBeInTheDocument()
  })

  it('masque la section sample quand sample_size=0', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: CAREER_FULL,
      sample_stats: { ...SAMPLE_FULL, sample_size: 0 },
      privacy_warning: null,
      auth_available: true,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="TargetPlayer" />)

    expect(screen.queryByText(/matchs joués ensemble/i)).not.toBeInTheDocument()
    // Career et identity restent
    expect(screen.getByText(/Carrière complète/i)).toBeInTheDocument()
    expect(screen.getByText('TargetPlayer')).toBeInTheDocument()
  })

  it('affiche le placeholder quand identity=null', () => {
    const profile: ExplorerTargetProfile = {
      identity: null,
      career_stats: null,
      sample_stats: null,
      privacy_warning: null,
      auth_available: false,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="UnknownPlayer" />)

    // Gamertag reste affiché
    expect(screen.getByText('UnknownPlayer')).toBeInTheDocument()
    // Message "Identité Spartan indisponible"
    expect(screen.getByText(/Identité Spartan indisponible/i)).toBeInTheDocument()
  })
})

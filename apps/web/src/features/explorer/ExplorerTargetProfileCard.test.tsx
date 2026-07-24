/**
 * Tests ExplorerTargetProfileCard (logique local-first, sans privacy) :
 *  - tout dispo (identity + career + sample)
 *  - no-tokens (auth_available=false → hint affiché, career masquée)
 *  - sample vide (sample_size=0 → section sample masquée)
 *  - identity=null → placeholder gamertag
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
    total_xp: 47820,
  },
  highest_csr: undefined,
  highest_lusr: undefined,
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
  highest_csr: 1523,
  highest_csr_all_time: 1600,
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
    // Section saisons TOUJOURS rendue avec placeholders titrés (visibilité des
    // manques), même sans season_csrs ni matches_per_season fournis.
    expect(screen.getByTestId('explorer-target-season-csr-empty')).toBeInTheDocument()
    expect(screen.getByTestId('explorer-target-season-matches')).toBeInTheDocument()
  })

  it('rend les sections enrichies (time-played, top médailles, CSR saison)', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: { ...CAREER_FULL, time_played_seconds: 90000 }, // 25h = 1j 1h
      sample_stats: SAMPLE_FULL,
      top_medals: [
        {
          medal_id: 622331684,
          label: 'Double frag',
          description: 'Tuez 2 ennemis.',
          image_url: '/static/medals/halo_infinite/622331684.png',
          total_count: 13118,
          match_count: 0,
        },
      ],
      season_csrs: [
        {
          playlist_id: 'p1',
          playlist_name: 'Ranked Arena',
          queue: '',
          input: '',
          current: { value: 1523, tier: 'Diamond', sub_tier: 3, measurement_matches_remaining: 0, placement_total: 0 },
          season: { value: 0, tier: '', sub_tier: 0, measurement_matches_remaining: 0, placement_total: 0 },
          all_time: { value: 0, tier: '', sub_tier: 0, measurement_matches_remaining: 0, placement_total: 0 },
        },
      ],
      auth_available: true,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="TargetPlayer" />)

    // Time played : KPI rendu (90000s = 1j 1h)
    expect(screen.getByTestId('explorer-target-time-played')).toBeInTheDocument()
    expect(screen.getByText('1j 1h')).toBeInTheDocument()
    // Top médailles : section + médaille
    expect(screen.getByTestId('explorer-target-medals')).toBeInTheDocument()
    expect(screen.getByText('Double frag')).toBeInTheDocument()
    // CSR saison : section + playlist + tier (traduit FR, locale défaut = fr)
    expect(screen.getByTestId('explorer-target-season-csr')).toBeInTheDocument()
    expect(screen.getByText('Ranked Arena')).toBeInTheDocument()
    expect(screen.getByText('Diamant III')).toBeInTheDocument()
    // Matchs par saison : colonne droite TOUJOURS rendue même sans matches_per_season
    // (placeholder titré) — jamais d'espace blanc silencieux à droite du bloc CSR.
    expect(screen.getByTestId('explorer-target-season-matches')).toBeInTheDocument()
    expect(screen.getByText('Aucun match par saison à afficher')).toBeInTheDocument()
  })

  it('ne rend jamais de bannière privacy (supprimée de l\'Explorer)', () => {
    // Même si l'API renvoyait une privacy (legacy), la carte ne la rend plus.
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: undefined,
      sample_stats: SAMPLE_FULL,
      privacy_warning: { level: 'full', message: 'Profil privé.' },
      auth_available: true,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="PrivatePlayer" />)

    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    expect(screen.queryByText('Profil privé.')).not.toBeInTheDocument()
    // Identité reste affichée
    expect(screen.getByText('PrivatePlayer')).toBeInTheDocument()
  })

  it('affiche le hint no-auth quand auth_available=false et career absente', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: undefined,
      sample_stats: SAMPLE_FULL,
      privacy_warning: undefined,
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

  it('masque la section sample quand sample_size=0 mais affiche une note explicite (V72-21)', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: CAREER_FULL,
      sample_stats: { ...SAMPLE_FULL, sample_size: 0 },
      privacy_warning: undefined,
      auth_available: true,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="TargetPlayer" />)

    expect(screen.queryByText(/matchs joués ensemble/i)).not.toBeInTheDocument()
    // Plus de disparition silencieuse : note discrète « aucun match en commun ».
    expect(screen.getByTestId('explorer-target-no-shared-matches')).toBeInTheDocument()
    expect(screen.getByText(/Aucun match en commun avec ce joueur/i)).toBeInTheDocument()
    // Career et identity restent
    expect(screen.getByText(/Carrière complète/i)).toBeInTheDocument()
    expect(screen.getByText('TargetPlayer')).toBeInTheDocument()
  })

  it('n\'affiche pas la note « aucun match en commun » sur une cible non résolue (identity=null, career absente)', () => {
    const profile: ExplorerTargetProfile = {
      identity: undefined,
      career_stats: undefined,
      sample_stats: undefined,
      privacy_warning: undefined,
      auth_available: false,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="UnknownPlayer" />)

    expect(screen.queryByTestId('explorer-target-no-shared-matches')).not.toBeInTheDocument()
  })

  it('affiche le placeholder quand identity=null', () => {
    const profile: ExplorerTargetProfile = {
      identity: undefined,
      career_stats: undefined,
      sample_stats: undefined,
      privacy_warning: undefined,
      auth_available: false,
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="UnknownPlayer" />)

    // Gamertag reste affiché
    expect(screen.getByText('UnknownPlayer')).toBeInTheDocument()
    // Message identité non disponible (reformulé, non alarmant)
    expect(screen.getByText(/Identité Spartan non disponible/i)).toBeInTheDocument()
  })
})

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
      // Cas heureux : tout a été fetché avec succès, rien à signaler.
      live_status: { identity: 'ok', career: 'ok', season_csrs: 'ok', seasons: 'ok', combat_live: 'ok' },
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
      // Cas heureux enrichi : carrière + médailles + CSR tous fetchés avec succès.
      live_status: { identity: 'ok', career: 'ok', season_csrs: 'ok', seasons: 'ok', combat_live: 'ok' },
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
      // auth dispo mais career_stats absent : le fetch a été tenté et n'a rien
      // renvoyé (rec nil) → failed, pas ok (career_stats ne serait jamais nil
      // si le service record avait réussi, cf. fetchTargetServiceRecord côté Go).
      live_status: { identity: 'ok', career: 'failed', season_csrs: 'ok', seasons: 'ok', combat_live: 'ok' },
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
      // Sans auth : identity reste ok (résolue localement, indépendante des
      // tokens) ; les 4 sections qui dépendent des tokens (career/CSR/saisons/
      // combat live) n'ont jamais été tentées → no_auth partout ailleurs.
      live_status: { identity: 'ok', career: 'no_auth', season_csrs: 'no_auth', seasons: 'no_auth', combat_live: 'no_auth' },
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
      // Carrière/identité fetchées avec succès ; sample_size=0 est un fait LOCAL
      // (aucun match commun), indépendant des statuts live.
      live_status: { identity: 'ok', career: 'ok', season_csrs: 'ok', seasons: 'ok', combat_live: 'ok' },
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
      // Cible totalement non résolue, sans auth : rien n'a jamais été tenté.
      live_status: { identity: 'no_auth', career: 'no_auth', season_csrs: 'no_auth', seasons: 'no_auth', combat_live: 'no_auth' },
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
      // Cible totalement non résolue, sans auth : rien n'a jamais été tenté.
      live_status: { identity: 'no_auth', career: 'no_auth', season_csrs: 'no_auth', seasons: 'no_auth', combat_live: 'no_auth' },
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="UnknownPlayer" />)

    // Gamertag reste affiché
    expect(screen.getByText('UnknownPlayer')).toBeInTheDocument()
    // Message identité non disponible (reformulé, non alarmant)
    expect(screen.getByText(/Identité Spartan non disponible/i)).toBeInTheDocument()
  })

  // --- Lot A3 (fin de la dégradation muette) : badges live_status par section ---

  it('affiche l\'en-tête « Carrière complète » + badge quand career_stats est vide mais live_status.career explique pourquoi', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: undefined,
      sample_stats: SAMPLE_FULL,
      auth_available: true,
      live_status: {
        identity: 'ok',
        career: 'failed',
        season_csrs: 'ok',
        seasons: 'ok',
        combat_live: 'ok',
      },
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="TargetPlayer" />)

    // Header rendu même sans données (avant A3 : section absente en silence).
    expect(screen.getByText(/Carrière complète/i)).toBeInTheDocument()
    expect(screen.getByTestId('explorer-live-status-badge-failed')).toBeInTheDocument()
    expect(screen.getByText('Données live indisponibles (erreur)')).toBeInTheDocument()
  })

  it('n\'affiche pas l\'en-tête « Carrière complète » quand live_status.career vaut ok malgré l\'absence de career_stats (rien à signaler)', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: undefined,
      sample_stats: SAMPLE_FULL,
      auth_available: false,
      // Robustesse défensive : même si le statut ne signale AUCUN problème
      // (ok), l'absence de career_stats ne doit jamais faire apparaître un
      // header vide — showCareerSection exige explicitement un statut != ok
      // pour se rabattre sur le header+badge (identity reste ok : résolue
      // localement, indépendante des tokens ; les autres sections restent
      // no_auth, cohérent avec auth_available=false, mais non exercées ici).
      live_status: { identity: 'ok', career: 'ok', season_csrs: 'no_auth', seasons: 'no_auth', combat_live: 'no_auth' },
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="TargetPlayer" />)

    // Le hint no-auth mentionne aussi "carrière complète" en texte libre (hors
    // heading) : on cible spécifiquement le TITRE de section (role heading) pour
    // ne pas confondre les deux — seul le heading doit être absent ici.
    expect(screen.queryByRole('heading', { name: /Carrière complète/i })).not.toBeInTheDocument()
  })

  it('ne duplique pas le badge carrière pour les médailles (même fetch, un seul badge affiché)', () => {
    // top_medals vide découle TOUJOURS du même échec que career_stats (même
    // fetch service record) : un seul badge (section Carrière) doit apparaître,
    // pas un second redondant à côté d'un bloc "Top médailles" vide.
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: undefined,
      sample_stats: SAMPLE_FULL,
      top_medals: [],
      auth_available: false,
      live_status: {
        identity: 'ok',
        career: 'no_auth',
        season_csrs: 'ok',
        seasons: 'ok',
        combat_live: 'no_auth',
      },
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="TargetPlayer" />)

    // Un seul badge no_auth au total (section Carrière) — pas de doublon médailles.
    expect(screen.getAllByTestId('explorer-live-status-badge-no_auth')).toHaveLength(1)
    expect(screen.queryByText(/Top médailles/i)).not.toBeInTheDocument()
  })

  it('affiche un badge sous le bandeau identité quand identity=null et live_status.identity != ok', () => {
    // Seul le statut identity est renseigné != ok (les autres = ok) pour isoler
    // le badge du bandeau identité de ceux des autres sections (career/seasons).
    const profile: ExplorerTargetProfile = {
      identity: undefined,
      career_stats: undefined,
      sample_stats: undefined,
      auth_available: false,
      live_status: {
        identity: 'no_auth',
        career: 'ok',
        season_csrs: 'ok',
        seasons: 'ok',
        combat_live: 'ok',
      },
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="UnknownPlayer" />)

    expect(screen.getByTestId('explorer-live-status-badge-no_auth')).toBeInTheDocument()
  })

  it('affiche le badge « Live partiel » sur la section saisons quand live_status.seasons = local_partial', () => {
    const profile: ExplorerTargetProfile = {
      identity: IDENTITY_FULL,
      career_stats: CAREER_FULL,
      sample_stats: SAMPLE_FULL,
      auth_available: true,
      live_status: {
        identity: 'ok',
        career: 'ok',
        season_csrs: 'ok',
        seasons: 'local_partial',
        combat_live: 'ok',
      },
    }
    renderWithProviders(<ExplorerTargetProfileCard profile={profile} gamertag="TargetPlayer" />)

    expect(screen.getByTestId('explorer-live-status-badge-local_partial')).toBeInTheDocument()
    expect(screen.getByText('Live partiel')).toBeInTheDocument()
  })
})

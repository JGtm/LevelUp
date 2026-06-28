/**
 * Tests ExplorerTargetIdentityBanner — rendu emblème / bannière / adornment
 * (parité visuelle Home) + cas dégradé identity=null.
 */
import { describe, expect, it, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomeSpartanIdentity, TitleSummary } from '@/lib/api/types'

import { ExplorerTargetIdentityBanner } from './ExplorerTargetIdentityBanner'

const IDENTITY_FULL: HomeSpartanIdentity = {
  banner_image_url: '/api/v1/assets/spartan/banner/halo_infinite/x.png',
  emblem_image_url: '/api/v1/assets/spartan/emblem/halo_infinite/x.png',
  spartan_id: 'ABCD',
  career_rank: {
    rank_number: 76,
    rank_title: 'Hero',
    current_xp: 47820,
    xp_for_next_rank: 50000,
    progress_pct: 95,
    is_max_rank: false,
    total_xp: 47820,
    adornment_image_url: '/api/v1/assets/spartan/adornment/halo_infinite/x.png',
  },
  highest_csr: undefined,
  highest_lusr: undefined,
}

function render(identity: HomeSpartanIdentity | null) {
  return renderWithProviders(
    <ExplorerTargetIdentityBanner
      identity={identity}
      gamertag="TargetPlayer"
      identityUnavailableLabel="Identité Spartan non disponible"
      identityUnavailableDescription="Connexion Halo requise."
    />,
  )
}

describe('ExplorerTargetIdentityBanner', () => {
  // Titre courant SANS spartan_customizer → bandeau normal (emblème <img>, pas la
  // synthèse). Sinon useCapability fail-open (availableTitles vide) → synthèse Halo 5.
  beforeEach(() => {
    useAppShellStore.setState({
      currentTitleSlug: 'halo_infinite',
      availableTitles: [
        {
          slug: 'halo_infinite',
          name: 'Halo Infinite',
          status: 'active',
          capabilities: [],
          is_default: true,
          effective_hp_to_kill: 225,
        } as unknown as TitleSummary,
      ],
    })
  })
  afterEach(() => {
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  })

  it('rend emblème, bannière, rang et adornment quand tout est fourni', () => {
    render(IDENTITY_FULL)

    expect(screen.getByTestId('explorer-target-emblem')).toHaveAttribute(
      'src',
      IDENTITY_FULL.emblem_image_url,
    )
    expect(screen.getByTestId('explorer-target-banner-image')).toBeInTheDocument()
    expect(screen.getByTestId('explorer-target-adornment-image')).toHaveAttribute(
      'src',
      IDENTITY_FULL.career_rank?.adornment_image_url,
    )
    expect(screen.getByText('Hero')).toBeInTheDocument()
    expect(screen.getByText('TargetPlayer')).toBeInTheDocument()
  })

  it("n'affiche pas l'adornment quand career_rank n'en porte pas", () => {
    render({
      ...IDENTITY_FULL,
      career_rank: { ...IDENTITY_FULL.career_rank!, adornment_image_url: undefined },
    })

    expect(screen.queryByTestId('explorer-target-adornment-image')).toBeNull()
    // Le reste reste rendu.
    expect(screen.getByTestId('explorer-target-emblem')).toBeInTheDocument()
    expect(screen.getByText('Hero')).toBeInTheDocument()
  })

  it('rend le placeholder (gamertag + label) quand identity=null', () => {
    render(null)

    expect(screen.getByText('TargetPlayer')).toBeInTheDocument()
    expect(screen.getByText('Identité Spartan non disponible')).toBeInTheDocument()
    expect(screen.queryByTestId('explorer-target-adornment-image')).toBeNull()
    expect(screen.queryByTestId('explorer-target-emblem')).toBeNull()
  })

  // Régression du fix Héros : la barre rang/XP ne doit JAMAIS disparaître quand
  // career_rank existe (cf. parité Home). Cas max rang inclus.
  it('affiche la barre de progression carrière dès que career_rank existe (rang normal)', () => {
    render(IDENTITY_FULL) // is_max_rank: false
    expect(screen.getByTestId('explorer-target-rank-progress-fill')).toBeInTheDocument()
    expect(screen.getByText(/Progression vers/)).toBeInTheDocument()
  })

  it('au rang max (Héros) : XP de carrière totale à gauche + un SEUL "Rang max"', () => {
    render({
      ...IDENTITY_FULL,
      career_rank: {
        ...IDENTITY_FULL.career_rank!,
        is_max_rank: true,
        progress_pct: 100,
        current_xp: 0,
        total_xp: 9319350,
      },
    })
    expect(screen.getByTestId('explorer-target-rank-progress-fill')).toBeInTheDocument()
    // Un seul "Rang max" (au bout de la barre composite), pas trois (cf. retour user).
    expect(screen.getAllByText('Rang max')).toHaveLength(1)
    // L'XP de carrière totale (« le grand nombre ») est affichée, pas « 0 XP ».
    expect(screen.getByText(/XP/)).toBeInTheDocument()
    expect(screen.queryByText('0 XP')).not.toBeInTheDocument()
  })

  it('localise les libellés des skill peaks (FR : Meilleur CSR / Meilleur LUSR)', () => {
    render({
      ...IDENTITY_FULL,
      highest_csr: { rating_value: 1500, tier_label: 'Onyx', measurement_matches_remaining: 0 },
      highest_lusr: { rating_value: 1600, tier_label: 'Diamant III', measurement_matches_remaining: 0 },
    })
    expect(screen.getByText('Meilleur CSR')).toBeInTheDocument()
    expect(screen.getByText('Meilleur LUSR')).toBeInTheDocument()
  })
})

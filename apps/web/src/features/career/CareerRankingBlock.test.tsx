/**
 * Tests composant — CareerRankingBlock (colonne LUSR title-aware).
 *
 * Invariants couverts :
 *   - Halo Infinite : les 4 groupes connus (arena_slayer/arena_objectif/btb/chaos)
 *     sont TOUJOURS rendus dans l'ordre déclaré, « Non classé » inclus (byte-identique).
 *   - Halo 5 : aucun groupe connu déclaré → on rend UNIQUEMENT le groupe présent
 *     dans les checkpoints (`h5_arena`), avec son libellé i18n « Arène ».
 */
import { beforeEach, describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TitleSummary } from '@/lib/api/types'
import type { CareerLusrSection } from '@/lib/api/types'
import { CareerRankingBlock } from './CareerRankingBlock'

// useCareerCSRs touche le réseau : on neutralise la colonne CSR (vide).
vi.mock('./queries', () => ({
  useCareerCSRs: () => ({ data: undefined }),
}))

const ALL_CAPS = ['ranked', 'lusr', 'career']

const HINF: TitleSummary = {
  slug: 'halo_infinite',
  name: 'Halo Infinite',
  status: 'active',
  capabilities: ALL_CAPS,
  is_default: true,
  effective_hp_to_kill: 225,
}

const HALO5: TitleSummary = {
  slug: 'halo_5',
  name: 'Halo 5',
  status: 'active',
  capabilities: ALL_CAPS,
  is_default: false,
  effective_hp_to_kill: 115,
}

function checkpoint(group: string): CareerLusrSection['checkpoints'][number] {
  return {
    recorded_at: '2026-06-01T00:00:00Z',
    rating_type: 'LUSR',
    rating_value: 1234,
    tier_label: 'Or III',
    playlist_group: group,
    playlist_name: 'Arena',
    badge_image_url: null,
  }
}

describe('CareerRankingBlock — colonne LUSR title-aware', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
  })

  it('Halo Infinite : rend les 4 groupes connus dans l\'ordre, « Non classé » inclus', () => {
    useAppShellStore.setState({ availableTitles: [HINF], currentTitleSlug: 'halo_infinite' })
    // Donnée partielle : seul arena_slayer a un checkpoint ; les 3 autres → « Non classé ».
    const lusr: CareerLusrSection = {
      current_rating: null,
      current_tier_label: null,
      current_playlist_group: null,
      trend_label: null,
      checkpoints: [checkpoint('arena_slayer')],
    }
    renderWithProviders(<CareerRankingBlock playerSlug="p" lusrData={lusr} />)

    // Les 4 libellés connus présents.
    expect(screen.getByText('Social · Assassin')).toBeInTheDocument()
    expect(screen.getByText('Social · Objectif')).toBeInTheDocument()
    expect(screen.getByText('Grande Équipe')).toBeInTheDocument()
    expect(screen.getByText('Chaos')).toBeInTheDocument()
    // 3 groupes non classés (FR « Non classé »).
    expect(screen.getAllByText('Non classé')).toHaveLength(3)
    // Pas de ligne h5 en HINF.
    expect(screen.queryByText('Arène')).not.toBeInTheDocument()
  })

  it('Halo 5 : rend uniquement le groupe présent (h5_arena → « Arène »)', () => {
    useAppShellStore.setState({ availableTitles: [HINF, HALO5], currentTitleSlug: 'halo_5' })
    const lusr: CareerLusrSection = {
      current_rating: 1234,
      current_tier_label: 'Or III',
      current_playlist_group: 'h5_arena',
      trend_label: null,
      checkpoints: [checkpoint('h5_arena')],
    }
    renderWithProviders(<CareerRankingBlock playerSlug="p" lusrData={lusr} />)

    // La ligne h5_arena s'affiche avec son libellé i18n.
    expect(screen.getByText('Arène')).toBeInTheDocument()
    // Aucun groupe HINF figé n'est rendu (pas de « Non classé » HINF parasite).
    expect(screen.queryByText('Social · Assassin')).not.toBeInTheDocument()
    expect(screen.queryByText('Grande Équipe')).not.toBeInTheDocument()
    expect(screen.queryByText('Chaos')).not.toBeInTheDocument()
    expect(screen.queryByText('Non classé')).not.toBeInTheDocument()
  })
})

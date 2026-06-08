/**
 * Tests d'ArcPresetPicker (Lot B — picker de presets d'arc).
 *
 * Vérifient : rendu de la liste des presets + aperçu (nb d'objectifs), et que
 * « Adopter » déclenche la mutation avec l'id du preset.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, cleanup, fireEvent } from '@testing-library/react'
import type { PresetArc } from '@/lib/prestige'

const adoptMutate = vi.fn()
const mockPresets: { current: PresetArc[]; isLoading: boolean; isError: boolean } = {
  current: [],
  isLoading: false,
  isError: false,
}

vi.mock('@/features/prestige/hooks', () => ({
  useArcPresets: () => ({
    data: { presets: mockPresets.current, count: mockPresets.current.length },
    isLoading: mockPresets.isLoading,
    isError: mockPresets.isError,
  }),
  useAdoptArcPreset: () => ({ mutate: adoptMutate, isPending: false }),
}))

import { ArcPresetPicker } from './ArcPresetPicker'

function preset(over: Partial<PresetArc>): PresetArc {
  return {
    id: 'p1',
    title_slug: 'halo_infinite',
    title_en: 'Spartan Ascension',
    title_fr: 'Ascension du Spartan',
    schema_version: 1,
    updated_at: '2026-01-01T00:00:00Z',
    steps: [
      { preset_arc_id: 'p1', position: 1, template_id: 't1', target_tier: 'normal' },
      { preset_arc_id: 'p1', position: 2, template_id: 't2', target_tier: 'heroic' },
    ],
    ...over,
  }
}

describe('ArcPresetPicker', () => {
  beforeEach(() => {
    cleanup()
    adoptMutate.mockClear()
    mockPresets.current = []
    mockPresets.isLoading = false
    mockPresets.isError = false
  })

  it('affiche les presets avec titre FR et aperçu du nombre d\'objectifs', () => {
    mockPresets.current = [preset({})]
    render(<ArcPresetPicker playerSlug="u1" titleSlug="halo_infinite" locale="fr" onClose={vi.fn()} />)

    expect(screen.getByText('Ascension du Spartan')).toBeInTheDocument()
    expect(screen.getByText(/2 objectifs/i)).toBeInTheDocument()
  })

  it('adopte un preset au clic sur « Adopter »', () => {
    mockPresets.current = [preset({ id: 'p-kda' })]
    render(<ArcPresetPicker playerSlug="u1" titleSlug="halo_infinite" locale="fr" onClose={vi.fn()} />)

    fireEvent.click(screen.getByRole('button', { name: 'Adopter' }))
    expect(adoptMutate).toHaveBeenCalledWith('p-kda', expect.objectContaining({ onSuccess: expect.any(Function) }))
  })

  it('affiche un état vide quand aucun preset', () => {
    mockPresets.current = []
    render(<ArcPresetPicker playerSlug="u1" titleSlug="halo_infinite" locale="fr" onClose={vi.fn()} />)
    expect(screen.getByText(/Aucun preset disponible/i)).toBeInTheDocument()
  })

  it('utilise le titre EN en locale en', () => {
    mockPresets.current = [preset({})]
    render(<ArcPresetPicker playerSlug="u1" titleSlug="halo_infinite" locale="en" onClose={vi.fn()} />)
    expect(screen.getByText('Spartan Ascension')).toBeInTheDocument()
    expect(screen.getByText(/2 objectives/i)).toBeInTheDocument()
  })
})

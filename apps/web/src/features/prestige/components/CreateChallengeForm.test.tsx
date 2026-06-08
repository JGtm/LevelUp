/**
 * Tests du retour UI cooldown dans CreateChallengeForm (Lot D, 2026-06-08).
 *
 * Vérifient :
 *   1. Mode hybride : un modèle dont la métrique est en cooldown affiche un
 *      badge « Dispo dans … » et n'est pas sélectionnable (aria-disabled).
 *   2. Le refus 429 (code cooldown_active) affiche un message lisible.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, cleanup, fireEvent } from '@testing-library/react'
import type { Template } from '@/lib/prestige'

const mockShellState = { locale: 'fr' as 'fr' | 'en' }
vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: <T,>(selector: (s: typeof mockShellState) => T) => selector(mockShellState),
}))

vi.mock('@/lib/i18n/fieldMappings', () => ({
  useAssetLabel: () => 'free',
}))

const mutate = vi.fn()
const mockCreate = { mutate, isPending: false, error: null as unknown }
const mockSuggested = {
  data: { templates: [] as Template[] },
  isLoading: false,
}
vi.mock('../hooks', () => ({
  useCreateChallenge: () => mockCreate,
  useSuggestedTemplates: () => mockSuggested,
}))

import { CreateChallengeForm } from './CreateChallengeForm'

function makeTemplate(over: Partial<Template>): Template {
  return {
    id: 'tpl-1',
    title_slug: 'halo_infinite',
    metric: 'FieldKDA',
    window_type: 'session',
    cadence: 'free',
    eval_type: 'threshold',
    mode_filter: 'universal',
    label_en: 'KDA challenge',
    label_fr: 'Défi KDA',
    normal_target: 1,
    heroic_target: 1.5,
    legendary_target: 2,
    mythic_target: 3,
    schema_version: 1,
    updated_at: '2026-01-01T00:00:00Z',
    ...over,
  }
}

describe('CreateChallengeForm — cooldown UI', () => {
  beforeEach(() => {
    cleanup()
    mockShellState.locale = 'fr'
    mockCreate.error = null
    mockSuggested.data = { templates: [] }
  })

  it('affiche un badge cooldown et désactive la sélection d\'un modèle en repos', () => {
    const future = new Date(Date.now() + 3 * 3_600_000).toISOString() // +3h
    mockSuggested.data = {
      templates: [makeTemplate({ id: 'cd', cooldown_ends_at: future })],
    }
    render(<CreateChallengeForm userId="u1" titleSlug="halo_infinite" />)

    expect(screen.getByText(/Dispo dans/i)).toBeInTheDocument()
    const item = screen.getByText('Défi KDA').closest('li')
    expect(item).toHaveAttribute('aria-disabled', 'true')

    // Clic sur un modèle en cooldown → pas de cible ajustée affichée (non sélectionné).
    fireEvent.click(item as HTMLElement)
    expect(screen.queryByText(/Cible ajustée/i)).not.toBeInTheDocument()
  })

  it('rend sélectionnable un modèle sans cooldown', () => {
    mockSuggested.data = { templates: [makeTemplate({ id: 'ok' })] }
    render(<CreateChallengeForm userId="u1" titleSlug="halo_infinite" />)

    expect(screen.queryByText(/Dispo dans/i)).not.toBeInTheDocument()
    const item = screen.getByText('Défi KDA').closest('li')
    fireEvent.click(item as HTMLElement)
    expect(screen.getByText(/Cible ajustée/i)).toBeInTheDocument()
  })

  it('affiche un message lisible sur refus cooldown (429)', () => {
    mockSuggested.data = { templates: [makeTemplate({ id: 'ok' })] }
    mockCreate.error = { code: 'cooldown_active', message: 'prestige: cooldown actif sur cette métrique' }
    render(<CreateChallengeForm userId="u1" titleSlug="halo_infinite" />)

    expect(screen.getByText(/Métrique en repos \(cooldown\)/i)).toBeInTheDocument()
  })
})

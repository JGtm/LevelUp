/**
 * Tests MilestonesGrid — états dégradés visibles (G2, 2026-07-26).
 *
 * Régression : /milestones répondait 500 sur halo_5 (Binder Error
 * condition_fr/condition_en). Le composant affichait bien un message d'erreur,
 * mais SANS le titre de section — sur une page longue, le bloc « Mes jalons »
 * semblait avoir purement disparu. Contrat vérifié : chargement / erreur / vide
 * gardent tous le titre de section, et le message d'erreur est localisé FR + EN.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { getAscensionText } from './i18n'

const mockShellState = {
  locale: 'fr' as 'fr' | 'en',
  currentTitleSlug: 'halo_5',
}

vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: <T,>(selector: (s: typeof mockShellState) => T) =>
    selector(mockShellState),
}))

type QueryState = {
  data?: { items: unknown[] }
  isLoading: boolean
  isError: boolean
}
const mockQuery: { current: QueryState } = {
  current: { isLoading: false, isError: false },
}

vi.mock('./queries', () => ({
  useMilestones: () => mockQuery.current,
}))

// Import APRÈS les vi.mock (hoistés par vitest, mais l'ordre reste explicite).
const { MilestonesGrid } = await import('./MilestonesGrid')

afterEach(() => {
  cleanup()
  mockShellState.locale = 'fr'
  mockQuery.current = { isLoading: false, isError: false }
})

describe('MilestonesGrid — états dégradés', () => {
  it('erreur (500 /milestones) : message localisé FR + titre de section conservé', () => {
    mockQuery.current = { isLoading: false, isError: true }
    render(<MilestonesGrid playerSlug="demo" />)

    const t = getAscensionText('fr')
    expect(screen.getByRole('alert').textContent).toBe(t.errorLoading)
    expect(
      screen.getByRole('heading', { name: t.milestonesSectionTitle }),
    ).toBeTruthy()
  })

  it('erreur : message localisé EN', () => {
    mockShellState.locale = 'en'
    mockQuery.current = { isLoading: false, isError: true }
    render(<MilestonesGrid playerSlug="demo" />)

    const t = getAscensionText('en')
    expect(screen.getByRole('alert').textContent).toBe(t.errorLoading)
    expect(
      screen.getByRole('heading', { name: t.milestonesSectionTitle }),
    ).toBeTruthy()
  })

  it('catalogue vide : message dédié + titre de section conservé', () => {
    mockQuery.current = { data: { items: [] }, isLoading: false, isError: false }
    render(<MilestonesGrid playerSlug="demo" />)

    const t = getAscensionText('fr')
    expect(screen.getByText(t.milestonesEmpty)).toBeTruthy()
    expect(
      screen.getByRole('heading', { name: t.milestonesSectionTitle }),
    ).toBeTruthy()
  })

  it('chargement : statut annoncé + titre de section conservé', () => {
    mockQuery.current = { isLoading: true, isError: false }
    render(<MilestonesGrid playerSlug="demo" />)

    const t = getAscensionText('fr')
    expect(screen.getByRole('status').textContent).toBe(t.loading)
    expect(
      screen.getByRole('heading', { name: t.milestonesSectionTitle }),
    ).toBeTruthy()
  })
})

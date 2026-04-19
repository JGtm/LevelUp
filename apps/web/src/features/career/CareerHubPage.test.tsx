/**
 * Tests composant — CareerHubPage (Sprint 55).
 *
 * Vérifie : tabs deep-linkables, redirect legacy citations, absence de top matchs / encounters
 * dans le hub, persistance du header.
 */
import { beforeEach, describe, it, expect, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { CareerHubPage } from './CareerHubPage'

const mockNavigate = vi.fn()
let activeTab: 'progression' | 'citations' = 'progression'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ playerSlug: 'test-player' }),
    useSearch: () => ({ tab: activeTab }),
  }
})

describe('CareerHubPage', () => {
  beforeEach(() => {
    activeTab = 'progression'
    mockNavigate.mockReset()
  })

  it('monte sans erreur', () => {
    const { container } = renderWithProviders(<CareerHubPage />)
    expect(container).toBeTruthy()
  })

  it("affiche le titre 'Carrière' dans le header", () => {
    renderWithProviders(<CareerHubPage />)
    expect(screen.getByText('Carrière')).toBeInTheDocument()
  })

  it("affiche les onglets 'Progression' et 'Citations'", () => {
    renderWithProviders(<CareerHubPage />)
    expect(screen.getByText('Progression')).toBeInTheDocument()
    expect(screen.getByText('Citations')).toBeInTheDocument()
  })

  it("l'onglet Progression est actif par défaut", () => {
    renderWithProviders(<CareerHubPage />)
    const progressionTab = screen.getByText('Progression')
    expect(progressionTab).toHaveClass('border-violet-600')
  })

  it("cliquer sur Citations navigue vers ?tab=citations", () => {
    renderWithProviders(<CareerHubPage />)
    fireEvent.click(screen.getByText('Citations'))
    expect(mockNavigate).toHaveBeenCalledWith(
      expect.objectContaining({ search: { tab: 'citations' } }),
    )
  })

  it("n'affiche pas de section 'top matchs' ou 'encounters' dans le hub", async () => {
    renderWithProviders(<CareerHubPage />)
    await waitFor(() => {
      expect(screen.queryByText(/top matchs/i)).not.toBeInTheDocument()
      expect(screen.queryByText(/Rencontres fréquentes/i)).not.toBeInTheDocument()
    })
  })

  it('le header persiste lors du changement de tab', () => {
    renderWithProviders(<CareerHubPage />)
    expect(screen.getByText('Carrière')).toBeInTheDocument()
    fireEvent.click(screen.getByText('Citations'))
    expect(screen.getByText('Carrière')).toBeInTheDocument()
  })
})

describe('CareerHubPage — onglet Citations', () => {
  beforeEach(() => {
    activeTab = 'citations'
    mockNavigate.mockReset()
  })

  it("affiche l'onglet Citations actif quand tab=citations", () => {
    renderWithProviders(<CareerHubPage />)
    const citationsTab = screen.getByText('Citations')
    expect(citationsTab).toHaveClass('border-violet-600')
  })
})

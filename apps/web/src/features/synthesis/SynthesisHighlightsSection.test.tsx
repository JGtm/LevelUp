/**
 * Tests — SynthesisHighlightsSection (Sprint 55 D5).
 */
import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { SynthesisHighlightsSection } from './SynthesisHighlightsSection'
import type { SynthesisHighlightsPreview } from '@/lib/api/types'

// TanStack Router : on remplace Link par un <a> simple pour éviter le contexte RouterProvider
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({ children, to }: { children?: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
  }
})

const playerSlug = 'test-player'

const fullPreview: SynthesisHighlightsPreview = {
  top_by_kills: [
    { match_id: 'aaaabbbbcccc1111', kills: 12, deaths: 3, kda: 4.0, outcome: 2, perf_score: 220 },
    { match_id: 'aaaabbbbcccc2222', kills: 10, deaths: 5, kda: 2.0, outcome: 2, perf_score: 190 },
  ],
  top_by_kda: [],
  worst_by_deaths: [
    { match_id: 'ddddeeeeffffg222', kills: 2, deaths: 15, kda: 0.13, outcome: 3, perf_score: 55 },
  ],
}

const fallbackPreview: SynthesisHighlightsPreview = {
  top_by_kills: [],
  top_by_kda: [
    { match_id: 'kkkkkkkkkkkk1111', kills: 8, deaths: 1, kda: 8.0, outcome: 2, perf_score: 300 },
  ],
  worst_by_deaths: [],
}

const emptyPreview: SynthesisHighlightsPreview = {
  top_by_kills: [],
  top_by_kda: [],
  worst_by_deaths: [],
}

function renderSection(preview: SynthesisHighlightsPreview) {
  return renderWithProviders(
    <SynthesisHighlightsSection highlights={preview} playerSlug={playerSlug} />,
  )
}

describe('SynthesisHighlightsSection', () => {
  describe('affichage normal', () => {
    it('monte sans erreur avec des données complètes', () => {
      const { container } = renderSection(fullPreview)
      expect(container).toBeTruthy()
    })

    it('affiche le titre "Meilleurs matchs"', () => {
      renderSection(fullPreview)
      expect(screen.getByText('Meilleurs matchs')).toBeInTheDocument()
    })

    it('affiche le titre "Matchs difficiles"', () => {
      renderSection(fullPreview)
      expect(screen.getByText('Matchs difficiles')).toBeInTheDocument()
    })

    it('affiche les kills du meilleur match', () => {
      renderSection(fullPreview)
      expect(screen.getByText('12')).toBeInTheDocument()
    })

    it('affiche le K/D du meilleur match', () => {
      renderSection(fullPreview)
      expect(screen.getByText('4.00')).toBeInTheDocument()
    })

    it('affiche le badge victoire (V) pour outcome=2', () => {
      renderSection(fullPreview)
      const badges = screen.getAllByText('V')
      expect(badges.length).toBeGreaterThan(0)
    })

    it('affiche le badge défaite (D) pour outcome=3', () => {
      renderSection(fullPreview)
      const badges = screen.getAllByText('D')
      expect(badges.length).toBeGreaterThan(0)
    })

    it('affiche un bouton qui navigue vers la page du match', () => {
      renderSection(fullPreview)
      // Phase 2a : le <Link> a été remplacé par un <button> qui appelle
      // useNavigateToMatch + persiste un MatchNavContext (groupe d'highlights).
      // Le label affiché est `${match_id.slice(0, 12)}…` — on cherche un
      // bouton dont le texte contient le préfixe d'un match_id de test.
      const buttons = screen.getAllByRole('button')
      expect(
        buttons.some((b) => b.textContent?.startsWith('aaaabbbb')),
      ).toBeTruthy()
    })

    it('affiche le score de perf si disponible', () => {
      renderSection(fullPreview)
      expect(screen.getByText('220')).toBeInTheDocument()
    })
  })

  describe('fallback top_by_kda', () => {
    it('utilise top_by_kda quand top_by_kills est vide', () => {
      renderSection(fallbackPreview)
      expect(screen.getByText('Meilleurs matchs')).toBeInTheDocument()
      // kda=8.0
      expect(screen.getByText('8.00')).toBeInTheDocument()
    })

    it('n\'affiche pas "Matchs difficiles" si worst_by_deaths est vide', () => {
      renderSection(fallbackPreview)
      expect(screen.queryByText('Matchs difficiles')).not.toBeInTheDocument()
    })
  })

  describe('état vide', () => {
    it('affiche un EmptyStateNotice si toutes les listes sont vides', () => {
      renderSection(emptyPreview)
      expect(screen.getByText(/Aucun match remarquable/i)).toBeInTheDocument()
    })

    it('n\'affiche pas de cartes de matchs si vide', () => {
      renderSection(emptyPreview)
      expect(screen.queryByText('Meilleurs matchs')).not.toBeInTheDocument()
      expect(screen.queryByText('Matchs difficiles')).not.toBeInTheDocument()
    })
  })

  describe('troncature match_id', () => {
    it('tronque le match_id à 12 caractères + "…"', () => {
      renderSection(fullPreview)
      // match_id='aaaabbbbcccc1111' → affiche 'aaaabbbbcccc…'
      expect(screen.getAllByText(/aaaabbbbcccc…/).length).toBeGreaterThan(0)
    })
  })
})

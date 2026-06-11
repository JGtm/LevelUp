/**
 * Tests AchievementsCareerSection — états loading, error, empty, et rendering
 * avec données. Mock du hook useAchievementsPage et du store locale.
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createElement, type ReactNode } from 'react'

import { AchievementsCareerSection } from './AchievementsCareerSection'
import type { AchievementsPageResponse } from '@/lib/api/types'

// Mock du store locale (par défaut FR)
vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: (selector: (s: { locale: 'fr' | 'en' }) => unknown) =>
    selector({ locale: 'fr' }),
}))

// Mock du hook
const mockHookReturn = {
  data: undefined as AchievementsPageResponse | undefined,
  isLoading: false,
  isError: false,
  refetch: vi.fn(),
}
vi.mock('./queries', () => ({
  useAchievementsPage: () => mockHookReturn,
}))

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return createElement(QueryClientProvider, { client }, children)
}

function reset(state: Partial<typeof mockHookReturn>) {
  mockHookReturn.data = undefined
  mockHookReturn.isLoading = false
  mockHookReturn.isError = false
  Object.assign(mockHookReturn, state)
}

describe('AchievementsCareerSection', () => {
  beforeEach(() => reset({}))

  it('rend un titre minimaliste pendant le loading', () => {
    reset({ isLoading: true })
    render(<AchievementsCareerSection playerSlug="jgtm" />, { wrapper })
    // Le titre est toujours affiché
    expect(screen.getByText('Succès Xbox')).toBeInTheDocument()
  })

  it('affiche un message d\'erreur avec bouton réessayer', () => {
    reset({ isError: true })
    render(<AchievementsCareerSection playerSlug="jgtm" />, { wrapper })
    expect(screen.getByText(/Erreur lors du chargement/)).toBeInTheDocument()
    expect(screen.getByText('Réessayer')).toBeInTheDocument()
  })

  it('affiche empty state quand total_count=0', () => {
    reset({
      data: {
        summary: {
          total_count: 0,
          unlocked_count: 0,
          total_gamerscore: 0,
          earned_gamerscore: 0,
          completion_pct: 0,
        },
        achievements: [],
      },
    })
    render(<AchievementsCareerSection playerSlug="jgtm" />, { wrapper })
    expect(screen.getByText(/Aucun succès en base/)).toBeInTheDocument()
    expect(screen.getByText(/levelup sync-achievements/)).toBeInTheDocument()
  })

  it('rend les KPIs et les cartes quand données présentes', () => {
    reset({
      data: {
        summary: {
          total_count: 100,
          unlocked_count: 42,
          total_gamerscore: 2000,
          earned_gamerscore: 800,
          completion_pct: 42.0,
        },
        achievements: [
          {
            achievement_id: 'a1',
            name_en: 'First Blood',
            name_fr: 'Premier sang',
            description_en: '',
            description_fr: '',
            gamerscore: 10,
            is_secret: false,
            unlocked: true,
          },
          {
            achievement_id: 'a2',
            name_en: 'Sharpshooter',
            name_fr: 'Tireur d\'élite',
            description_en: '',
            description_fr: '',
            gamerscore: 25,
            is_secret: false,
            unlocked: false,
          },
        ],
      },
    })
    render(<AchievementsCareerSection playerSlug="jgtm" />, { wrapper })
    // KPI inline
    expect(screen.getByText('42 / 100')).toBeInTheDocument()
    expect(screen.getByText('800 / 2000 G')).toBeInTheDocument()
    expect(screen.getByText('42.0 %')).toBeInTheDocument()
    // Cards : noms FR (locale mock = 'fr')
    expect(screen.getByText('Premier sang')).toBeInTheDocument()
    expect(screen.getByText('Tireur d\'élite')).toBeInTheDocument()
  })

  it('rend toutes les cartes (pas de cap : refacto 2026-05 → scroll vertical)', () => {
    // Refacto 2026-05 : la sidebar utilise overflow-y-auto + maxHeight 640px
    // au lieu d'un cap VISIBLE_LIMIT. Toutes les cartes sont rendues, le DOM
    // gère la visibilité via le scroll natif.
    const items = Array.from({ length: 50 }, (_, i) => ({
      achievement_id: `a${i}`,
      name_en: `Ach ${i}`,
      name_fr: `Succès ${i}`,
      description_en: '',
      description_fr: '',
      gamerscore: i,
      is_secret: false,
      unlocked: false,
    }))
    reset({
      data: {
        summary: {
          total_count: 50,
          unlocked_count: 0,
          total_gamerscore: 0,
          earned_gamerscore: 0,
          completion_pct: 0,
        },
        achievements: items,
      },
    })
    const { container } = render(<AchievementsCareerSection playerSlug="jgtm" />, { wrapper })
    const cards = container.querySelectorAll('[role="listitem"]')
    expect(cards.length).toBe(50)
  })

  it('filtre par catégorie en layout sidebar (select visible si catégories présentes)', () => {
    reset({
      data: {
        summary: {
          total_count: 3,
          unlocked_count: 0,
          total_gamerscore: 60,
          earned_gamerscore: 0,
          completion_pct: 0,
        },
        achievements: [
          {
            achievement_id: 'a1',
            name_en: 'Clocking In',
            name_fr: 'Pointage',
            description_en: '',
            description_fr: '',
            gamerscore: 10,
            is_secret: false,
            unlocked: false,
            category: 'multiplayer',
          },
          {
            achievement_id: 'a2',
            name_en: 'Zeta',
            name_fr: 'Zêta',
            description_en: '',
            description_fr: '',
            gamerscore: 20,
            is_secret: false,
            unlocked: false,
            category: 'campaign',
          },
          {
            achievement_id: 'a3',
            name_en: 'Get the Popcorn',
            name_fr: 'Sortez le pop-corn',
            description_en: '',
            description_fr: '',
            gamerscore: 30,
            is_secret: false,
            unlocked: false,
            category: 'other',
          },
        ],
      },
    })
    render(<AchievementsCareerSection playerSlug="jgtm" layout="sidebar" />, { wrapper })

    // Multijoueur par défaut : seule la carte MP est visible
    const categorySelect = screen.getByDisplayValue('Multijoueur')
    expect(screen.getByText('Pointage')).toBeInTheDocument()
    expect(screen.queryByText('Zêta')).not.toBeInTheDocument()
    expect(screen.queryByText('Sortez le pop-corn')).not.toBeInTheDocument()

    fireEvent.change(categorySelect, { target: { value: 'all' } })
    expect(screen.getByText('Pointage')).toBeInTheDocument()
    expect(screen.getByText('Zêta')).toBeInTheDocument()
    expect(screen.getByText('Sortez le pop-corn')).toBeInTheDocument()

    fireEvent.change(categorySelect, { target: { value: 'campaign' } })
    expect(screen.queryByText('Pointage')).not.toBeInTheDocument()
    expect(screen.getByText('Zêta')).toBeInTheDocument()
  })

  it('masque le select catégorie quand aucune entrée n\'a de catégorie (titre sans mapping)', () => {
    reset({
      data: {
        summary: {
          total_count: 1,
          unlocked_count: 0,
          total_gamerscore: 10,
          earned_gamerscore: 0,
          completion_pct: 0,
        },
        achievements: [
          {
            achievement_id: 'a1',
            name_en: 'No Category',
            name_fr: 'Sans catégorie',
            description_en: '',
            description_fr: '',
            gamerscore: 10,
            is_secret: false,
            unlocked: false,
          },
        ],
      },
    })
    render(<AchievementsCareerSection playerSlug="jgtm" layout="sidebar" />, { wrapper })
    expect(screen.queryByDisplayValue('Multijoueur')).not.toBeInTheDocument()
    // Le défaut "multiplayer" ne doit pas filtrer un titre sans mapping : la carte reste visible
    expect(screen.getByText('Sans catégorie')).toBeInTheDocument()
    // Les filtres statut + tri date restent présents
    expect(screen.getByDisplayValue('Tous')).toBeInTheDocument()
    expect(screen.getByDisplayValue('Défaut')).toBeInTheDocument()
  })
})

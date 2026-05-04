/**
 * Tests AchievementsCareerSection — états loading, error, empty, et rendering
 * avec données. Mock du hook useAchievementsPage et du store locale.
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
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

  it('limite le nombre de cartes affichées à VISIBLE_LIMIT', () => {
    // Génère 50 entrées — doit n'en rendre que 30 visibles.
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
    expect(cards.length).toBe(30)
  })
})

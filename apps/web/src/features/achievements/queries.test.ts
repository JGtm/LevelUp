/**
 * Tests useAchievementsPage — vérifie le query key et l'URL d'appel.
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { createElement } from 'react'

import { queryKeys } from '@/lib/query/keys'
import { api } from '@/lib/api/client'
import { useAchievementsPage } from './queries'
import type { AchievementsPageResponse } from '@/lib/api/types'

vi.mock('@/lib/api/client', () => ({
  api: { get: vi.fn() },
  // Le hook lit désormais le titre courant (useAppShellStore -> soloFilterStore ->
  // getApiTitleSlug) : le mock doit exposer ces exports sinon l'import du store échoue.
  getApiTitleSlug: () => 'halo_infinite',
  setApiTitleSlug: vi.fn(),
  setApiLocale: vi.fn(),
}))

const mockResponse: AchievementsPageResponse = {
  summary: {
    total_count: 100,
    unlocked_count: 42,
    total_gamerscore: 2000,
    earned_gamerscore: 800,
    completion_pct: 42,
  },
  achievements: [
    {
      achievement_id: 'a',
      name_en: 'First',
      name_fr: 'Premier',
      description_en: 'Desc',
      description_fr: 'Description',
      gamerscore: 10,
      is_secret: false,
      unlocked: true,
      unlocked_at: '2026-04-15T10:00:00Z',
    },
  ],
}

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return createElement(QueryClientProvider, { client }, children)
}

describe('useAchievementsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('appelle GET /pages/achievements avec le bon slug', async () => {
    const apiGet = vi.mocked(api.get)
    apiGet.mockResolvedValueOnce(mockResponse)

    const { result } = renderHook(() => useAchievementsPage('jgtm'), { wrapper })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(apiGet).toHaveBeenCalledWith('/players/jgtm/pages/achievements')
    expect(result.current.data).toEqual(mockResponse)
  })

  it('utilise le query key queryKeys.achievements(slug, titleSlug)', () => {
    const key = queryKeys.achievements('test-player', 'halo_infinite')
    expect(key).toEqual(['achievements', 'test-player', 'halo_infinite'])
  })

  it('désactivé quand le slug est vide', () => {
    const apiGet = vi.mocked(api.get)
    const { result } = renderHook(() => useAchievementsPage(''), { wrapper })

    expect(result.current.fetchStatus).toBe('idle')
    expect(apiGet).not.toHaveBeenCalled()
  })
})

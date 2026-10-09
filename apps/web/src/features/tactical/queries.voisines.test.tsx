/**
 * Changer de lecture sans recharger : une réponse du plan RANGE ses voisines (même lecture de la
 * base côté serveur) sous les clés exactes de leurs questions, et l'intention de changer de lecture
 * précharge, une par une, celles que le cache n'a pas encore.
 */
import { describe, expect, it, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import type { TacticalRaster } from '@/lib/api/types'
import { usePrechargementDesLectures, useTacticalRaster, type ParamsDuRaster } from './queries'

const post = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, post: (path: string, body: unknown) => post(path, body) } }
})

function lecture(question: string, voisines: string[] = []): TacticalRaster {
  return {
    map_id: 'streets',
    question,
    qui: 'moi',
    bornes: { min_x: 0, max_x: 10, min_y: 0, max_y: 10, valide: true },
    pas_m: 2,
    echelle: { p50: 1, p95: 2, borne: 2, n_cellules: 0, symetrique: false },
    cellules: [],
    matchs_filtres: 3,
    matchs_retenus: 3,
    matchs_victoire: 0,
    matchs_defaite: 0,
    points_ignores: 0,
    zones: question === voisines[0] ? undefined : [],
    voisines: voisines.map((q) => lecture(q)),
  }
}

/** Le serveur : chaque question rend sa famille (positions ; sidecars ; isole seule). */
const FAMILLES = [['morts', 'kills', 'solde', 'gagne'], ['temps', 'routes'], ['isole']]
function repondre(_path: string, corps: { question: string }) {
  const famille = FAMILLES.find((f) => f.includes(corps.question)) ?? [corps.question]
  return Promise.resolve(lecture(corps.question, famille.filter((q) => q !== corps.question)))
}

const PARAMS = (question: string): ParamsDuRaster => ({
  match_ids: ['m1', 'm2', 'm3'],
  coequipiers: [],
  question,
  qui: 'moi',
  spawn: undefined,
})

function monter<T>(hook: () => T) {
  // Le client de l'app garde une entrée sans observateur plusieurs minutes (`gcTime`) ; celui des
  // tests par défaut la jette aussitôt, ce qui effacerait les voisines rangées avant leur lecture.
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 60_000 } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return renderHook(hook, { wrapper })
}

describe('useTacticalRaster — les voisines rangées', () => {
  it('une seule requête, puis les quatre lectures des positions servies par le cache', async () => {
    post.mockReset().mockImplementation(repondre)
    let question = 'morts'
    const { result, rerender } = monter(() => useTacticalRaster('JGtm', 'streets', PARAMS(question)))
    await waitFor(() => expect(result.current.data?.question).toBe('morts'))
    expect(post).toHaveBeenCalledTimes(1)

    for (const q of ['kills', 'solde', 'gagne']) {
      question = q
      rerender()
      expect(result.current.data?.question).toBe(q)
      expect(result.current.isPlaceholderData).toBe(false)
      expect(result.current.data?.zones).toEqual([])
    }
    expect(post).toHaveBeenCalledTimes(1)

    question = 'temps'
    rerender()
    await waitFor(() => expect(result.current.data?.question).toBe('temps'))
    expect(post).toHaveBeenCalledTimes(2)
  })
})

describe('usePrechargementDesLectures — à l’intention de changer de lecture', () => {
  it('ne demande que ce qui manque, une famille à la fois, et une seule fois', async () => {
    post.mockReset().mockImplementation(repondre)
    const questions = ['morts', 'kills', 'solde', 'gagne', 'temps', 'routes', 'isole']
    const { result } = monter(() => ({
      lecture: useTacticalRaster('JGtm', 'streets', PARAMS('morts')),
      precharger: usePrechargementDesLectures('JGtm', 'streets', PARAMS('morts'), questions),
    }))
    await waitFor(() => expect(result.current.lecture.data?.question).toBe('morts'))

    act(() => result.current.precharger())
    act(() => result.current.precharger())
    await waitFor(() => expect(post).toHaveBeenCalledTimes(3))
    const demandees = post.mock.calls.map(([, corps]) => (corps as { question: string }).question)
    expect(demandees).toEqual(['morts', 'temps', 'isole'])

    // Tout est en cache : une nouvelle intention ne demande rien.
    await new Promise((r) => setTimeout(r, 0))
    act(() => result.current.precharger())
    await new Promise((r) => setTimeout(r, 0))
    expect(post).toHaveBeenCalledTimes(3)
  })

  it('périmètre non résolu : aucun préchargement', () => {
    post.mockReset().mockImplementation(repondre)
    const { result } = monter(() =>
      usePrechargementDesLectures('JGtm', 'streets', { ...PARAMS('morts'), match_ids: null }, ['temps']),
    )
    act(() => result.current())
    expect(post).not.toHaveBeenCalled()
  })
})

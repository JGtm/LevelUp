/**
 * useCompositionOptions — la liste branchée sur les vraies lectures : amis déclarés d'abord, bots
 * exclus, profils suivis dans l'annuaire, et des tableaux MÉMOÏSÉS sur les réponses (un tableau
 * neuf à chaque rendu recalculait, sur la page Tactique, la composition puis les paramètres et
 * l'empreinte de chaque lecture).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { useAppShellStore } from '@/stores/appShellStore'
import { createTestQueryClient } from '@/test/render-utils'

import { useCompositionOptions } from './useCompositionOptions'

const get = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return { ...actual, api: { ...actual.api, get: (path: string) => get(path) } }
})

const RENCONTRES = {
  teammates: [
    { gamertag: '343 Bot', xuid: 'bid(1.0)', match_count: 50, as_teammate: 50, as_enemy: 0, avg_kda: null },
    { gamertag: 'Inconnu', xuid: 'x-inconnu', match_count: 40, as_teammate: 40, as_enemy: 0, avg_kda: null },
    { gamertag: 'Ami', xuid: 'x-ami', match_count: 30, as_teammate: 30, as_enemy: 0, avg_kda: null },
  ],
  enemies: [],
  total: 3,
}

function repondre(amis: string[], groupes: () => Promise<unknown> = async () => []) {
  get.mockImplementation(async (path: string) => {
    if (path.includes('career/encounters')) return RENCONTRES
    if (path.endsWith('/pages/squad')) return { top_teammates: AVEC_AMIS }
    if (path.endsWith('/friends')) return { xuid: 'x-jgtm', gamertags: amis, can_edit: true }
    if (path.includes('/squads')) return { squads: [], count: 0 }
    if (path === '/groups') return groupes()
    throw new Error(`appel inattendu : ${path}`)
  })
}

/** Le top des coéquipiers des matchs avec amis : un ami hors des joueurs croisés, et un bot. */
const AVEC_AMIS = [
  { gamertag: 'Lointain', xuid: 'x-lointain', games_together: 7 },
  { gamertag: '343 Bot', xuid: 'bid(9.0)', games_together: 70 },
]

const MEMBRE_DE_GROUPE = { gamertag: 'Membre', xuid: 'x-membre', role: 'member', joined_at: '2026-01-01T00:00:00Z' }
const GROUPE = { id: 'g1', name: 'Groupe', members: [MEMBRE_DE_GROUPE] }

function monter() {
  const client = createTestQueryClient()
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return renderHook(() => useCompositionOptions('JGtm'), { wrapper })
}

afterEach(() => {
  get.mockReset()
  useAppShellStore.setState({ availablePlayers: [], linkedHaloIdentity: null })
})

describe('useCompositionOptions', () => {
  it('amis déclarés seuls, bots exclus, profils suivis dans l’annuaire, xuid du joueur', async () => {
    repondre(['Ami', 'Profil', 'Lointain'])
    useAppShellStore.setState({
      availablePlayers: [
        { gamertag: 'JGtm', xuid: 'x-jgtm', player_slug: 'JGtm', is_demo: false, sync_enabled: true, waypoint_player: 'JGtm' },
        { gamertag: 'Profil', xuid: 'x-profil', player_slug: 'Profil', is_demo: false, sync_enabled: true, waypoint_player: 'Profil' },
      ],
    })
    const { result } = monter()
    await waitFor(() => expect(result.current.chargees).toBe(true))
    expect(result.current.options.map((o) => o.gamertag)).toEqual(['Ami', 'Lointain', 'Profil'])
    expect(result.current.annuaire.map((o) => o.gamertag).sort()).toEqual(['Ami', 'Inconnu', 'Lointain', 'Profil'])
    expect(result.current.joueurXuid).toBe('x-jgtm')
  })

  it('« chargées » ATTEND les groupes : un membre connu par eux seuls n’est jamais déclaré introuvable trop tôt', async () => {
    let livrer: (v: unknown) => void = () => {}
    repondre(['Ami'], () => new Promise((r) => (livrer = r)))
    useAppShellStore.setState({ linkedHaloIdentity: { gamertag: 'JGtm', xuid: 'x-jgtm' } })
    const { result } = monter()
    await waitFor(() => expect(get).toHaveBeenCalledWith('/groups'))
    await waitFor(() => expect(get).toHaveBeenCalledWith(expect.stringContaining('/squads')))
    // Les trois autres lectures ont répondu ; les groupes, non.
    await new Promise((r) => setTimeout(r, 20))
    expect(result.current.chargees).toBe(false)
    livrer([GROUPE])
    await waitFor(() => expect(result.current.chargees).toBe(true))
    expect(result.current.annuaire.map((o) => o.gamertag)).toContain('Membre')
  })

  it('une source en ÉCHEC compte comme répondue, et l’échec est journalisé', async () => {
    const avert = vi.spyOn(console, 'warn').mockImplementation(() => {})
    try {
      repondre(['Ami'], async () => {
        throw new Error('groupes indisponibles')
      })
      useAppShellStore.setState({ linkedHaloIdentity: { gamertag: 'JGtm', xuid: 'x-jgtm' } })
      const { result } = monter()
      await waitFor(() => expect(result.current.chargees).toBe(true))
      expect(avert).toHaveBeenCalledWith(
        expect.stringContaining('[composition]'),
        expect.objectContaining({ source: 'groupes' }),
      )
    } finally {
      avert.mockRestore()
    }
  })

  it('rend le MÊME tableau d’un rendu à l’autre sur les mêmes réponses', async () => {
    repondre(['Ami'])
    const { result, rerender } = monter()
    await waitFor(() => expect(result.current.chargees).toBe(true))
    const { options, annuaire } = result.current
    rerender()
    expect(result.current.options).toBe(options)
    expect(result.current.annuaire).toBe(annuaire)
  })
})

/**
 * Câblage — useKillSourceIcons : le référentiel des pictogrammes d'arme du titre.
 *
 * Ce qu'ils protègent : une requête par titre et par langue (la langue est dans la clé de requête) ;
 * la table est rangée par weapon_key ; pendant le chargement et en cas d'erreur la table est vide, sans
 * exception — les lignes qui s'en servent s'affichent sans pictogramme ; un titre vide ne demande rien.
 */
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

import type { KillSourceIconEntry } from '@/lib/api/types'
import { KNOWN_LOCALES } from '@/lib/i18n/generated/locales'
import { queryKeys } from '@/lib/query/keys'

const hoisted = vi.hoisted(() => ({
  reply: [] as unknown,
  fail: false,
}))

vi.mock('@/lib/api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/client')>()),
  api: {
    get: vi.fn(async () => {
      if (hoisted.fail) throw new Error('réseau')
      return hoisted.reply
    }),
  },
}))

import { api } from '@/lib/api/client'
import { useKillSourceIcons } from './useKillSourceIcons'

const [LOCALE_A, LOCALE_B] = KNOWN_LOCALES

const entry = (key: string, label = key): KillSourceIconEntry => ({
  key,
  label,
  image_url: `/static/${key}.png`,
  tinted: true,
})

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { client, wrapper }
}

beforeEach(() => {
  hoisted.reply = []
  hoisted.fail = false
  vi.mocked(api.get).mockClear()
})

describe('useKillSourceIcons', () => {
  it('vide pendant le chargement, puis rangé par weapon_key', async () => {
    hoisted.reply = [entry('hinf_hydra', 'Hydra'), entry('hinf_br75', 'BR75')]
    const { wrapper } = setup()
    const { result } = renderHook(() => useKillSourceIcons('halo_infinite', LOCALE_A), { wrapper })
    expect(result.current.size).toBe(0)
    await waitFor(() => expect(result.current.size).toBe(2))
    expect(result.current.get('hinf_hydra')).toEqual({ imageUrl: '/static/hinf_hydra.png', tinted: true, label: 'Hydra' })
    expect(api.get).toHaveBeenCalledWith('/assets/halo_infinite/kill-sources')
  })

  it('en erreur : table vide, sans exception', async () => {
    hoisted.fail = true
    const { wrapper } = setup()
    const { result } = renderHook(() => useKillSourceIcons('halo_infinite', LOCALE_A), { wrapper })
    await waitFor(() => expect(api.get).toHaveBeenCalled())
    expect(result.current.size).toBe(0)
  })

  it('une réponse nulle donne une table vide', async () => {
    hoisted.reply = null
    const { wrapper } = setup()
    const { result } = renderHook(() => useKillSourceIcons('halo_infinite', LOCALE_A), { wrapper })
    await waitFor(() => expect(api.get).toHaveBeenCalled())
    expect(result.current.size).toBe(0)
  })

  it('la langue entre dans la clé de requête : deux langues, deux entrées de cache', async () => {
    hoisted.reply = [entry('hinf_br75')]
    const { client, wrapper } = setup()
    const a = renderHook(() => useKillSourceIcons('halo_infinite', LOCALE_A), { wrapper })
    const b = renderHook(() => useKillSourceIcons('halo_infinite', LOCALE_B), { wrapper })
    await waitFor(() => expect(a.result.current.size).toBe(1))
    await waitFor(() => expect(b.result.current.size).toBe(1))
    expect(api.get).toHaveBeenCalledTimes(2)
    expect(client.getQueryData(queryKeys.killSourceIcons('halo_infinite', LOCALE_A))).toBeDefined()
    expect(client.getQueryData(queryKeys.killSourceIcons('halo_infinite', LOCALE_B))).toBeDefined()
    expect(queryKeys.killSourceIcons('halo_infinite', LOCALE_A)).not.toEqual(
      queryKeys.killSourceIcons('halo_infinite', LOCALE_B),
    )
  })

  it('un titre vide ne demande rien', () => {
    const { wrapper } = setup()
    const { result } = renderHook(() => useKillSourceIcons('', LOCALE_A), { wrapper })
    expect(result.current.size).toBe(0)
    expect(api.get).not.toHaveBeenCalled()
  })
})

/**
 * Câblage — useChatterLines : la table des répliques se télécharge UNE fois par titre, seulement quand le film porte au moins
 * une réplique ; elle s'annule au démontage ; un fichier introuvable laisse le rejeu intact (aucune réplique).
 *
 * Chaque test prend son propre titre : la table chargée est gardée par titre pour toute la page.
 */
import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { testReplayDoc } from '../test/testDoc'

const hoisted = vi.hoisted(() => ({
  reply: null as unknown,
  fail: false,
  signals: [] as AbortSignal[],
  resolvers: [] as Array<() => void>,
  hold: false,
}))

vi.mock('@/lib/api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/client')>()),
  api: {
    getStatic: vi.fn(async (_path: string, options?: { signal?: AbortSignal }) => {
      if (options?.signal) hoisted.signals.push(options.signal)
      if (hoisted.hold) await new Promise<void>((resolve) => hoisted.resolvers.push(resolve))
      if (hoisted.fail) throw new Error('introuvable')
      return hoisted.reply
    }),
  },
}))

import { api } from '@/lib/api/client'
import { persistPreference } from '../settings/replayPreferences'
import { useChatterFeedLines, useChatterLines, useChatterTable } from './useChatterLines'

const TABLE = {
  version: 1,
  sons: {
    '0000000a': { texte: { fr: 'Ennemi aperçu !', en: 'Enemy there.' }, audio: ['fr'] },
  },
}

let slugCounter = 0
const nextSlug = () => `titre_${(slugCounter += 1)}`

function docOf(slug: string, withChatter = true) {
  return testReplayDoc({
    titleSlug: slug,
    chatter: withChatter
      ? [
          { t: 10, player: 'A', sound: '0000000a', name: 'seeingenemynearby' },
          { t: 20, player: 'B', sound: '0000000a', name: 'seeingenemybelow' },
        ]
      : [],
  })
}

beforeEach(() => {
  hoisted.reply = TABLE
  hoisted.fail = false
  hoisted.hold = false
  hoisted.signals = []
  hoisted.resolvers = []
  vi.mocked(api.getStatic).mockClear()
  localStorage.clear()
})

afterEach(() => {
  localStorage.clear()
})

describe('useChatterLines', () => {
  it('ne demande rien à un film sans réplique', () => {
    const { result } = renderHook(() => useChatterLines(docOf(nextSlug(), false), 'fr'))
    expect(result.current).toEqual([])
    expect(api.getStatic).not.toHaveBeenCalled()
  })

  it('télécharge la table du titre, puis rend les répliques dans la langue demandée', async () => {
    const slug = nextSlug()
    const doc = docOf(slug)
    const { result } = renderHook(() => useChatterLines(doc, 'fr'))
    expect(result.current).toEqual([]) // la table n'est pas encore arrivée
    await waitFor(() => expect(result.current).toHaveLength(2))
    expect(result.current[0]).toMatchObject({ text: 'Ennemi aperçu !', voice: 'repliques/fr/0000000a', family: 'enemy', count: 2 })
    expect(vi.mocked(api.getStatic).mock.calls[0][0]).toBe(`/static/sounds/${slug}/repliques/repliques.json`)
  })

  it("suit la langue de l'interface sans retélécharger : l'anglais a son texte, et la voix versée n'est que française", async () => {
    const doc = docOf(nextSlug())
    const { result, rerender } = renderHook(({ l }) => useChatterLines(doc, l), { initialProps: { l: 'fr' as 'fr' | 'en' } })
    await waitFor(() => expect(result.current).toHaveLength(2))
    rerender({ l: 'en' })
    expect(result.current[0]).toMatchObject({ text: 'Enemy there.', voice: null })
    expect(api.getStatic).toHaveBeenCalledTimes(1)
  })

  it("garde la table par titre : un second montage ne la redemande pas", async () => {
    const doc = docOf(nextSlug())
    const first = renderHook(() => useChatterLines(doc, 'fr'))
    await waitFor(() => expect(first.result.current).toHaveLength(2))
    const second = renderHook(() => useChatterLines(doc, 'fr'))
    expect(second.result.current).toHaveLength(2)
    expect(api.getStatic).toHaveBeenCalledTimes(1)
  })

  it('un fichier introuvable laisse la table vide, sans erreur', async () => {
    hoisted.fail = true
    const { result } = renderHook(() => useChatterLines(docOf(nextSlug()), 'fr'))
    await waitFor(() => expect(api.getStatic).toHaveBeenCalled())
    await act(async () => {})
    expect(result.current).toEqual([])
  })

  it("annule la requête au démontage : la réponse arrivée après coup n'écrit rien", async () => {
    hoisted.hold = true
    const { unmount, result } = renderHook(() => useChatterLines(docOf(nextSlug()), 'fr'))
    await waitFor(() => expect(hoisted.signals).toHaveLength(1))
    unmount()
    expect(hoisted.signals[0].aborted).toBe(true)
    await act(async () => {
      hoisted.resolvers.forEach((r) => r())
    })
    expect(result.current).toEqual([])
  })
})

describe('useChatterTable', () => {
  it('rend la table vide quand le film n’en a pas besoin', () => {
    const { result } = renderHook(() => useChatterTable(nextSlug(), false))
    expect(result.current.size).toBe(0)
    expect(api.getStatic).not.toHaveBeenCalled()
  })
})

describe('useChatterFeedLines', () => {
  it('rend les répliques qui passent les filtres quand le fil les accueille', async () => {
    const doc = docOf(nextSlug())
    const { result } = renderHook(() => useChatterFeedLines(doc, 'fr'))
    await waitFor(() => expect(result.current).toHaveLength(2))
  })

  it('rend rien quand « Répliques dans le fil » est éteint', async () => {
    persistPreference('replay-show-chatter-feed', 'false')
    const doc = docOf(nextSlug())
    const { result } = renderHook(() => useChatterFeedLines(doc, 'fr'))
    await waitFor(() => expect(api.getStatic).toHaveBeenCalled())
    await act(async () => {})
    expect(result.current).toEqual([])
  })

  it('retire une famille éteinte', async () => {
    persistPreference('replay-chatter-family-enemy', 'false')
    const doc = docOf(nextSlug())
    const { result } = renderHook(() => useChatterFeedLines(doc, 'fr'))
    await waitFor(() => expect(api.getStatic).toHaveBeenCalled())
    await act(async () => {})
    expect(result.current).toEqual([])
  })

  it("suit un changement de réglage fait ailleurs (la clé est partagée par abonnement)", async () => {
    const doc = docOf(nextSlug())
    const { result } = renderHook(() => useChatterFeedLines(doc, 'fr'))
    await waitFor(() => expect(result.current).toHaveLength(2))
    act(() => persistPreference('replay-chatter-frequency', 'rare'))
    // les deux répliques disent la même phrase : deux fois, donc « rares » (au plus deux)
    expect(result.current).toHaveLength(2)
    act(() => persistPreference('replay-chatter-frequency', 'noFrequent'))
    expect(result.current).toHaveLength(2)
    act(() => persistPreference('replay-chatter-family-enemy', 'false'))
    expect(result.current).toHaveLength(0)
  })
})

describe('une seule requête par titre, partagée entre les lecteurs', () => {
  it('quatre lecteurs montés ensemble (carte, voix, panneau, fil) : une seule requête', async () => {
    hoisted.hold = true
    const doc = docOf(nextSlug())
    const readers = [
      renderHook(() => useChatterLines(doc, 'fr')),
      renderHook(() => useChatterLines(doc, 'fr')),
      renderHook(() => useChatterFeedLines(doc, 'fr')),
      renderHook(() => useChatterLines(doc, 'fr')),
    ]
    await waitFor(() => expect(api.getStatic).toHaveBeenCalled())
    expect(api.getStatic).toHaveBeenCalledTimes(1)
    await act(async () => {
      hoisted.resolvers.forEach((r) => r())
    })
    for (const r of readers) await waitFor(() => expect(r.result.current).toHaveLength(2))
    expect(api.getStatic).toHaveBeenCalledTimes(1)
  })

  it("un lecteur qui se démonte n'annule pas la requête des autres ; le dernier l'annule", async () => {
    hoisted.hold = true
    const doc = docOf(nextSlug())
    const a = renderHook(() => useChatterLines(doc, 'fr'))
    const b = renderHook(() => useChatterLines(doc, 'fr'))
    await waitFor(() => expect(hoisted.signals).toHaveLength(1))
    a.unmount()
    expect(hoisted.signals[0].aborted).toBe(false)
    await act(async () => {
      hoisted.resolvers.forEach((r) => r())
    })
    await waitFor(() => expect(b.result.current).toHaveLength(2))
    // le dernier lecteur part avant la réponse : la requête s'annule
    const slug = nextSlug()
    const c = renderHook(() => useChatterLines(docOf(slug), 'fr'))
    await waitFor(() => expect(hoisted.signals).toHaveLength(2))
    c.unmount()
    expect(hoisted.signals[1].aborted).toBe(true)
  })

  it('un échec n’est pas mémorisé : un nouveau montage retente', async () => {
    hoisted.fail = true
    const doc = docOf(nextSlug())
    const first = renderHook(() => useChatterLines(doc, 'fr'))
    await waitFor(() => expect(api.getStatic).toHaveBeenCalledTimes(1))
    await act(async () => {})
    expect(first.result.current).toEqual([])
    first.unmount()
    hoisted.fail = false
    const second = renderHook(() => useChatterLines(doc, 'fr'))
    await waitFor(() => expect(second.result.current).toHaveLength(2))
    expect(api.getStatic).toHaveBeenCalledTimes(2)
  })
})

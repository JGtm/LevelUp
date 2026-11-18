/**
 * usePlayerInstant.test.tsx — LES SÉRIES SE LISENT UNE FOIS, JAMAIS PAR IMAGE.
 *
 * La fiche suit la lecture : à chaque image publiée le hook rend une valeur différente, sans relire le document.
 * Les deux lectures coûteuses (`readInstantDocSeries`, par document et lobby ; `readPlayerInstantSources`, par
 * joueur) sont comptées : une image de plus ne les rappelle pas, un autre joueur ne rappelle que la seconde, la
 * langue relit la seconde seulement.
 */
import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { vitalityPresence } from '@/lib/replay/rosterLogic'

import {
  INSTANT_ICONS, instantDoc, instantFeed, instantPlayer, instantPlayers,
} from '../test/instantFixture'
import { readInstantDocSeries, readPlayerInstantSources } from '../model/playerInstantSources'
import { usePlayerInstant, type PlayerInstantInput } from './usePlayerInstant'

vi.mock('../model/playerInstantSources', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../model/playerInstantSources')>()
  return {
    ...actual,
    readInstantDocSeries: vi.fn(actual.readInstantDocSeries),
    readPlayerInstantSources: vi.fn(actual.readPlayerInstantSources),
  }
})

const DOC = instantDoc()
const PLAYERS = instantPlayers(DOC)
const FEED = instantFeed()
const PRESENCE = vitalityPresence(DOC)

function input(over: Partial<PlayerInstantInput> = {}): PlayerInstantInput {
  return {
    doc: DOC, player: instantPlayer(PLAYERS, 'A'), players: PLAYERS, feed: FEED, presence: PRESENCE,
    matchStartFrame: null, weaponIcons: INSTANT_ICONS, locale: 'fr', ...over,
  }
}

beforeEach(() => {
  vi.mocked(readInstantDocSeries).mockClear()
  vi.mocked(readPlayerInstantSources).mockClear()
})

describe('usePlayerInstant', () => {
  it('rend la fiche de l’image lue, et la suit sans relire les séries', () => {
    const { result, rerender } = renderHook(({ frame }) => usePlayerInstant(input(), frame), { initialProps: { frame: 100 } })
    expect(result.current.identity).toMatchObject({ kills: 3 })
    expect(result.current.since?.respawns).toBe(0)
    rerender({ frame: 399 })
    expect(result.current.identity).toMatchObject({ kills: 4 })
    expect(result.current.since?.respawns).toBe(1)
    rerender({ frame: 160 })
    expect(result.current.since?.currentLifeMs).toBeNull()
    // Trois images lues, une seule lecture de chaque série.
    expect(readInstantDocSeries).toHaveBeenCalledTimes(1)
    expect(readPlayerInstantSources).toHaveBeenCalledTimes(1)
  })

  it('une même image relue rend la même fiche, sans nouvelle lecture', () => {
    const { result, rerender } = renderHook(({ frame }) => usePlayerInstant(input(), frame), { initialProps: { frame: 100 } })
    const first = result.current
    rerender({ frame: 100 })
    expect(result.current).toBe(first)
    expect(readPlayerInstantSources).toHaveBeenCalledTimes(1)
  })

  it('un autre joueur : la lecture du joueur est refaite, celle du document non', () => {
    const { result, rerender } = renderHook(({ who }) => usePlayerInstant(input({ player: instantPlayer(PLAYERS, who) }), 399), {
      initialProps: { who: 'A' },
    })
    expect(result.current.shots).not.toBeNull()
    rerender({ who: 'E' })
    expect(result.current.shots).toBeNull()
    expect(readInstantDocSeries).toHaveBeenCalledTimes(1)
    expect(readPlayerInstantSources).toHaveBeenCalledTimes(2)
  })

  it('la langue relit les noms du joueur, pas le document', () => {
    const { result, rerender } = renderHook(({ locale }) => usePlayerInstant(input({ locale }), 399), {
      initialProps: { locale: 'fr' as 'fr' | 'en' },
    })
    expect(result.current.grenadesThrown[0].name).toBe('Fragmentation')
    rerender({ locale: 'en' })
    expect(result.current.grenadesThrown[0].name).toBe('Frag')
    expect(readInstantDocSeries).toHaveBeenCalledTimes(1)
  })
})

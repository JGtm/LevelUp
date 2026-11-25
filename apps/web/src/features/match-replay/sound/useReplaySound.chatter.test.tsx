/**
 * Câblage — les voix des répliques dans le son du rejeu : chargées À LA DEMANDE (la fenêtre des quelques secondes à venir),
 * jamais en bloc, jamais tant que le son est coupé, que la voix est éteinte ou que la langue n'en a pas ; une voix pas prête
 * à l'instant de sa réplique est sautée ; tout s'annule au démontage et au changement de match.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/features/settings/queries', () => ({ useSettings: () => ({ data: undefined }) }))

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import { type FakeContext, flushAudio, installFakeAudio, okAudioResponse } from '../test/fakeAudio'
import { testReplayDoc } from '../test/testDoc'
import { SOUND_MAX_SPEED } from './replaySoundCursor'
import type { ReplaySoundEvent } from './replaySoundVariants'
import { useReplaySound } from './useReplaySound'

let ctx: FakeContext
let fetchMock: ReturnType<typeof vi.fn>

/** Les demandes de voix de réplique faites au réseau. */
const voiceFetches = () => fetchMock.mock.calls.map((c) => String(c[0])).filter((u) => u.includes('/repliques/'))

beforeEach(() => {
  const fake = installFakeAudio()
  ctx = fake.ctx
  fetchMock = fake.fetchMock
  localStorage.clear()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  localStorage.clear()
})

const voice = (ms: number, id: string): ReplaySoundEvent => ({ ms, stem: `repliques/fr/${id}`, chatter: true })
const URL_OF = (id: string) => `/static/sounds/halo_infinite/repliques/fr/${id}.ogg`

const VOICES = [voice(3_000, 'a'), voice(9_000, 'b'), voice(20_000, 'c')]
const doc = testReplayDoc({ frameIntervalMs: 100 })

function mount(voices: readonly ReplaySoundEvent[] = VOICES, speed = 1) {
  return renderHook(
    ({ s, d, v }: { s: number; d: typeof doc; v: readonly ReplaySoundEvent[] }) =>
      useReplaySound(d, [], s, { allegiance: NO_ALLEGIANCE, endMatch: null, locale: undefined, chatterVoices: v }),
    { initialProps: { s: speed, d: doc, v: voices } },
  )
}

async function activate(result: { current: ReturnType<typeof useReplaySound> }) {
  act(() => result.current.toggle())
  await act(async () => { await flushAudio() })
}

/** Un `fetch` dont les voix de réplique n'arrivent que quand le test le décide ; chaque demande note son signal. */
function holdVoices() {
  const held: Array<{ url: string; signal: AbortSignal; release: () => void }> = []
  const base = fetchMock
  const hold = vi.fn((url: string, init?: { signal?: AbortSignal }) => {
    if (!String(url).includes('/repliques/')) return (base as unknown as (u: string) => Promise<unknown>)(url)
    return new Promise((resolve, reject) => {
      const signal = init?.signal as AbortSignal
      signal.addEventListener('abort', () => reject(new DOMException('annulé', 'AbortError')))
      held.push({ url: String(url), signal, release: () => resolve(okAudioResponse(1)) })
    })
  })
  vi.stubGlobal('fetch', hold)
  return held
}

describe('aucune voix n’est demandée quand elle ne peut pas sonner', () => {
  it('le son est coupé : zéro requête de voix, même en lisant tout le match', () => {
    const { result } = mount()
    for (const ms of [0, 2_900, 3_100, 8_900, 19_900]) act(() => result.current.tick(ms))
    expect(voiceFetches()).toEqual([])
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('le réglage « voix des répliques » est éteint : zéro requête de voix, son activé', async () => {
    const { result } = mount()
    act(() => result.current.toggleCategory('voice'))
    await activate(result)
    for (const ms of [0, 2_900, 3_100, 8_900, 19_900]) act(() => result.current.tick(ms))
    expect(voiceFetches()).toEqual([])
  })

  it('la langue affichée n’a pas de voix (aucun événement de voix) : zéro requête de voix', async () => {
    const { result } = mount([])
    await activate(result)
    for (const ms of [0, 2_900, 3_100, 8_900]) act(() => result.current.tick(ms))
    expect(voiceFetches()).toEqual([])
  })

  it('l’activation du son ne précharge AUCUNE voix, même si le match en porte', async () => {
    const { result } = mount()
    await activate(result)
    expect(voiceFetches()).toEqual([])
  })
})

describe('chargement à la demande', () => {
  it('seules les voix de la fenêtre d’avance sont demandées, au fil de la lecture', async () => {
    const { result } = mount()
    await activate(result)
    act(() => result.current.tick(2_900)) // fenêtre ]2 900 ; 6 900] : la voix de 3 000
    await act(async () => { await flushAudio() })
    expect(voiceFetches()).toEqual([URL_OF('a')])
    act(() => result.current.tick(5_000)) // ]5 000 ; 9 000] : la voix de 9 000
    await act(async () => { await flushAudio() })
    expect(voiceFetches()).toEqual([URL_OF('a'), URL_OF('b')])
    expect(voiceFetches()).not.toContain(URL_OF('c')) // 20 000 est loin
  })

  it('une voix prête à l’instant de sa réplique est jouée, une seule fois', async () => {
    const { result } = mount()
    await activate(result)
    act(() => result.current.tick(2_900)) // recalage, demande de la voix de 3 000 (un saut de plus d'une seconde recale sans rien jouer)
    await act(async () => { await flushAudio() })
    const avant = ctx.sources.length
    act(() => result.current.tick(3_050))
    expect(ctx.sources.length).toBe(avant + 1)
    act(() => result.current.tick(3_200))
    expect(ctx.sources.length).toBe(avant + 1)
  })

  it('une voix pas encore prête à l’instant de sa réplique est sautée : pas de son en retard', async () => {
    const held = holdVoices()
    const { result } = mount()
    await activate(result)
    act(() => result.current.tick(2_900)) // la voix de 3 000 est demandée, mais n'arrive pas
    expect(held).toHaveLength(1)
    const avant = ctx.sources.length
    act(() => result.current.tick(3_050)) // l'instant passe sans que la voix soit prête
    expect(ctx.sources.length).toBe(avant)
    await act(async () => { held[0].release(); await flushAudio() }) // elle arrive après coup
    act(() => result.current.tick(3_200))
    expect(ctx.sources.length).toBe(avant) // jamais jouée en retard
  })

  it('au-delà de 2× rien ne se demande ni ne se joue', async () => {
    const { result } = mount(VOICES, SOUND_MAX_SPEED * 2)
    await activate(result)
    for (const ms of [2_000, 3_050, 4_000]) act(() => result.current.tick(ms))
    expect(voiceFetches()).toEqual([])
    expect(ctx.sources).toHaveLength(0)
  })
})

describe('annulations', () => {
  it('démontage pendant un chargement : la requête est annulée, rien ne se passe ensuite', async () => {
    const held = holdVoices()
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const { result, unmount } = mount()
    await activate(result)
    act(() => result.current.tick(2_900))
    expect(held).toHaveLength(1)
    const sources = ctx.sources.length
    unmount()
    expect(held[0].signal.aborted).toBe(true)
    expect(ctx.closed).toBe(1)
    await act(async () => { held[0].release(); await flushAudio() })
    expect(ctx.sources.length).toBe(sources)
    expect(warn).not.toHaveBeenCalled()
  })

  it('changement de match : les chargements en cours sont annulés', async () => {
    const held = holdVoices()
    const { result, rerender } = mount()
    await activate(result)
    act(() => result.current.tick(2_900))
    expect(held).toHaveLength(1)
    rerender({ s: 1, d: testReplayDoc({ frameIntervalMs: 100, matchId: 'autre' }), v: VOICES })
    expect(held[0].signal.aborted).toBe(true)
  })

  it('couper le son annule les chargements en cours et libère les voix', async () => {
    const held = holdVoices()
    const { result } = mount()
    await activate(result)
    act(() => result.current.tick(2_900))
    act(() => result.current.toggle()) // son coupé
    expect(held[0].signal.aborted).toBe(true)
  })

  it('éteindre la voix des répliques annule les chargements en cours', async () => {
    const held = holdVoices()
    const { result } = mount()
    await activate(result)
    act(() => result.current.tick(2_900))
    act(() => result.current.toggleCategory('voice'))
    expect(held[0].signal.aborted).toBe(true)
  })
})

describe('le reste de la piste', () => {
  it('les voix suffisent à offrir la commande du son', () => {
    expect(mount().result.current.available).toBe(true)
  })

  it('la catégorie « Voix » éteinte reste éteinte au prochain rejeu (préférence retenue)', () => {
    const first = mount()
    act(() => first.result.current.toggleCategory('voice'))
    first.unmount()
    expect(mount().result.current.categories.voice).toBe(false)
  })

  it('l’export classe les stems de voix dans la famille « voix » et garde les événements dans la piste', () => {
    const track = mount().result.current.exportTrack()
    expect(track.families.voice).toContain('repliques/fr/a')
    expect(track.timeline.some((e) => e.stem === 'repliques/fr/a')).toBe(true)
  })
})

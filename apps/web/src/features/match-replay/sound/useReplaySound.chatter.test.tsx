/**
 * Câblage — les voix des répliques dans la piste du son : elles entrent dans la piste, se préchargent, se jouent à leur
 * instant quand le son est activé, et se retirent avec la catégorie « Voix » ; le son coupé par défaut ne télécharge rien.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/features/settings/queries', () => ({ useSettings: () => ({ data: undefined }) }))

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import { type FakeContext, flushAudio, installFakeAudio } from '../test/fakeAudio'
import { testReplayDoc } from '../test/testDoc'
import { SOUND_MAX_SPEED } from './replaySoundCursor'
import type { ReplaySoundEvent } from './replaySoundVariants'
import { useReplaySound } from './useReplaySound'

let ctx: FakeContext
let fetchMock: ReturnType<typeof vi.fn>

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

const VOICE: ReplaySoundEvent = { ms: 3_000, stem: 'repliques/fr/0000000a', chatter: true }

function mount(voices: readonly ReplaySoundEvent[] = [VOICE], speed = 1) {
  const doc = testReplayDoc({ frameIntervalMs: 100 })
  return renderHook(
    ({ s }: { s: number }) =>
      useReplaySound(doc, [], s, { allegiance: NO_ALLEGIANCE, endMatch: null, locale: undefined, chatterVoices: voices }),
    { initialProps: { s: speed } },
  )
}

describe('les voix des répliques dans la piste du son', () => {
  it('suffisent à offrir la commande du son, et ne téléchargent rien tant que le son est coupé', () => {
    const { result } = mount()
    expect(result.current.available).toBe(true)
    act(() => result.current.tick(3_100))
    expect(ctx.sources).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('se préchargent avec la piste à l’activation, puis jouent à leur instant, une seule fois', async () => {
    const { result } = mount()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(fetchMock.mock.calls.map((c) => String(c[0]))).toContain('/static/sounds/halo_infinite/repliques/fr/0000000a.ogg')
    act(() => result.current.tick(2_900)) // recalage avant la voix
    act(() => result.current.tick(3_050)) // la voix passe ici
    expect(ctx.sources).toHaveLength(1)
    act(() => result.current.tick(3_150))
    expect(ctx.sources).toHaveLength(1)
  })

  it('la catégorie « Voix » éteinte retire la voix de la piste : rien ne se télécharge ni ne sonne', async () => {
    const { result } = mount()
    act(() => result.current.toggleCategory('voice'))
    expect(result.current.categories.voice).toBe(false)
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(fetchMock.mock.calls.map((c) => String(c[0]))).not.toContain('/static/sounds/halo_infinite/repliques/fr/0000000a.ogg')
    act(() => result.current.tick(2_900))
    act(() => result.current.tick(3_050))
    expect(ctx.sources).toHaveLength(0)
  })

  it('la catégorie « Voix » éteinte reste éteinte au prochain rejeu (préférence retenue)', () => {
    const first = mount()
    act(() => first.result.current.toggleCategory('voice'))
    first.unmount()
    const second = mount()
    expect(second.result.current.categories.voice).toBe(false)
  })

  it('au-delà de 2× la voix se tait comme le reste du son, sans rien rattraper au retour', async () => {
    const { result, rerender } = mount([VOICE], SOUND_MAX_SPEED * 2)
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(result.current.mutedBySpeed).toBe(true)
    act(() => result.current.tick(2_900))
    act(() => result.current.tick(3_050))
    expect(ctx.sources).toHaveLength(0)
    rerender({ s: 1 })
    act(() => result.current.tick(3_150))
    expect(ctx.sources).toHaveLength(0)
  })

  it("range les stems de voix dans la famille « voix » de l'export", () => {
    const { result } = mount()
    const track = result.current.exportTrack()
    expect(track.families.voice).toContain('repliques/fr/0000000a')
    expect(track.timeline.some((e) => e.stem === 'repliques/fr/0000000a')).toBe(true)
  })
})

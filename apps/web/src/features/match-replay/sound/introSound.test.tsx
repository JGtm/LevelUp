/**
 * introSound.test.tsx — LA MUSIQUE D'INTRO (item 7 du backlog, décision D-8), côté son.
 *
 * Le DÉCLENCHEMENT (quand la lecture part du préambule) se teste côté lecture
 * (`hooks/useReplayPlayback.intro.test.tsx`). Ici, ce que le son en fait : l'extrait passe par
 * la voie ordinaire du lecteur, donc par ses silences — son coupé, vitesse au-delà de
 * SOUND_MAX_SPEED — et il est préchargé avec la piste, sans jamais entrer dans l'export.
 */
import { act, renderHook } from '@testing-library/react'

vi.mock('@/features/settings/queries', () => ({ useSettings: () => ({ data: undefined }) }))
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import type { ReplayKill } from '../model/killFeedLogic'
import { type FakeContext, flushAudio, installFakeAudio, okAudioResponse } from '../test/fakeAudio'
import { testReplayDoc } from '../test/testDoc'
import { INTRO_MUSIC_STEM } from './introSound'
import { SOUND_MAX_SPEED } from './replaySoundCursor'
import { useReplaySound, type ReplaySoundContext } from './useReplaySound'

const NO_CONTEXT: ReplaySoundContext = { allegiance: NO_ALLEGIANCE, endMatch: null, locale: undefined }
const INTRO_URL = `/static/sounds/halo_infinite/${INTRO_MUSIC_STEM}.wav`
/** La durée factice de l'extrait (1 octet = 0,1 s dans le faux décodeur) : c'est elle qu'on lit. */
const INTRO_S = 3.3

let ctx: FakeContext
let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  const fake = installFakeAudio()
  ctx = fake.ctx
  fetchMock = fake.fetchMock
  fetchMock.mockImplementation((url: string) => Promise.resolve(okAudioResponse(url === INTRO_URL ? INTRO_S : 1.2)))
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function kill(): ReplayKill {
  return {
    replayMs: 2_000, medals: [], tMs: 2_000, xuid: 'K', ally: true, teamID: 0,
    weaponKey: 'hinf_br75', weaponLabel: 'BR75', weaponImageUrl: '', weaponTinted: false,
    assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null, assistDamagePct: null,
    victimXuid: 'V', victimGamertag: 'Victime', victimTeamID: 1,
  }
}

function mount(speed = 1) {
  const doc = testReplayDoc({
    frameIntervalMs: 100,
    tracks: [
      { slot: 1, team: -1, xuid: 'K', points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 10, y: 0 }], startFrame: 0, endFrame: 100 },
      { slot: 2, team: -1, xuid: 'V', points: [{ t: 0, x: 5, y: 5 }, { t: 20, x: 5, y: 4 }], startFrame: 0, endFrame: 20 },
    ],
  })
  const kills = [kill()]
  return renderHook(({ s }: { s: number }) => useReplaySound(doc, kills, s, NO_CONTEXT), { initialProps: { s: speed } })
}

const introSources = () => ctx.sources.filter((s) => s.buffer?.duration === INTRO_S)

describe('useReplaySound — la musique d’intro', () => {
  it('son activé, à 1× : l’extrait est préchargé avec la piste et part une fois', async () => {
    const { result } = mount()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(fetchMock.mock.calls.map((c) => String(c[0]))).toContain(INTRO_URL)
    act(() => result.current.intro())
    expect(introSources()).toHaveLength(1)
  })

  // DA-9 / A4.9 : premier « Lecture » après un rechargement. Le lecteur naît DANS ce geste
  // (`wake`), l'extrait n'est pas encore décodé quand `intro` est appelée juste après.
  it('premier « Lecture » après un rechargement : l’extrait part quand même, une fois décodé', async () => {
    localStorage.setItem('replay-sound-on', 'true')
    const { result } = mount()
    act(() => {
      result.current.wake()
      result.current.intro()
    })
    expect(introSources()).toHaveLength(0)
    await act(async () => { await flushAudio() })
    expect(introSources()).toHaveLength(1)
    localStorage.clear()
  })

  it('à SOUND_MAX_SPEED tout juste, elle part encore', async () => {
    const { result } = mount(SOUND_MAX_SPEED)
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    act(() => result.current.intro())
    expect(introSources()).toHaveLength(1)
  })

  it('son coupé : silence, et aucun contexte ni téléchargement au passage', () => {
    const { result } = mount()
    act(() => result.current.intro())
    expect(ctx.sources).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('au-delà de SOUND_MAX_SPEED : silence, comme le reste du son', async () => {
    const { result } = mount(SOUND_MAX_SPEED * 2)
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    act(() => result.current.intro())
    expect(ctx.sources).toHaveLength(0)
  })

  it('absente de l’export : ni dans la piste ni dans la conclusion, rangée en musique', async () => {
    const { result } = mount()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    const track = result.current.exportTrack()
    expect(track.timeline.map((e) => e.stem)).not.toContain(INTRO_MUSIC_STEM)
    expect(track.endMatchStems).not.toContain(INTRO_MUSIC_STEM)
    expect(track.families.music).toContain(INTRO_MUSIC_STEM)
  })
})

/**
 * useReplaySound.pause.test.tsx — LA PAUSE ÉTEINT TOUT, ET UN DÉPLACEMENT À L'ARRÊT NE RALLUME PAS.
 *
 * Constaté sur le terrain (contrôles v8, 2026-10-21) : les sons de véhicule continuaient une fois
 * le rejeu mis en pause. Deux causes, deux tests :
 *  1. un déplacement à l'arrêt (glissé de frise, pas d'image, saut vers un repère) bat le son ET
 *     y réconciliait les MOTEURS, qui redémarraient leur boucle sans que plus rien ne l'éteigne ;
 *  2. la pause n'éteignait que les moteurs et les rafales tenues, pas les autres sons en vol.
 *
 * Les moteurs sont doublés (leur lecteur a ses propres tests) : ce qui se vérifie ici est le
 * CÂBLAGE — quand le hook demande aux moteurs de suivre, et quand il leur demande de se taire.
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/features/settings/queries', () => ({ useSettings: () => ({ data: undefined }) }))

const engine = vi.hoisted(() => ({
  stems: [] as string[],
  attach: vi.fn(),
  sync: vi.fn(),
  stop: vi.fn(),
  plansForExport: vi.fn(() => []),
}))
vi.mock('./useReplayEngineSound', () => ({ useReplayEngineSound: () => engine }))

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import type { ReplayKill } from '../model/killFeedLogic'
import { type FakeContext, flushAudio, installFakeAudio } from '../test/fakeAudio'
import { testReplayDoc } from '../test/testDoc'
import { VOLUME_RAMP_S } from './replayAudio'
import { useReplaySound, type ReplaySoundContext } from './useReplaySound'

const NO_CONTEXT: ReplaySoundContext = { allegiance: NO_ALLEGIANCE, endMatch: null, locale: undefined }

let ctx: FakeContext

beforeEach(() => {
  ctx = installFakeAudio().ctx
  engine.attach.mockClear()
  engine.sync.mockClear()
  engine.stop.mockClear()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function kill(over: Partial<ReplayKill> = {}): ReplayKill {
  return {
    replayMs: 2_000, medals: [],
    tMs: 2_000, xuid: 'K', teamID: 0,
    weaponKey: 'hinf_br75', weaponLabel: 'BR75', weaponImageUrl: '', weaponTinted: false,
    assistState: '', assistGamertag: '', assistTeamID: null,
    killerDamagePct: null, assistDamagePct: null,
    victimXuid: 'V', victimGamertag: 'Victime',
    ...over,
  }
}

function docWithCouple() {
  return testReplayDoc({
    frameIntervalMs: 100,
    tracks: [
      { slot: 1, team: -1, xuid: 'K', points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 10, y: 0 }], startFrame: 0, endFrame: 100 },
      { slot: 2, team: -1, xuid: 'V', points: [{ t: 0, x: 5, y: 5 }, { t: 20, x: 5, y: 4 }], startFrame: 0, endFrame: 20 },
    ],
  })
}

async function mountOn() {
  const doc = docWithCouple()
  const kills = [kill()]
  const hook = renderHook(() => useReplaySound(doc, kills, 1, NO_CONTEXT))
  act(() => hook.result.current.toggle())
  await act(async () => { await flushAudio() })
  return hook
}

describe('useReplaySound — les moteurs ne suivent que la lecture', () => {
  it('en lecture, un battement fait suivre les moteurs', async () => {
    const { result } = await mountOn()
    act(() => result.current.setTransportPlaying(true))
    engine.sync.mockClear()
    act(() => result.current.tick(1_900))
    expect(engine.sync).toHaveBeenCalledWith(1_900)
  })

  it('À L ARRÊT, un déplacement (seek) ne réveille AUCUN moteur', async () => {
    const { result } = await mountOn()
    act(() => result.current.setTransportPlaying(false))
    engine.sync.mockClear()
    act(() => result.current.seek(5_000))
    expect(engine.sync).not.toHaveBeenCalled()
    expect(engine.stop).toHaveBeenCalled()
  })

  it('À L ARRÊT, un pas d image (tick) ne réveille aucun moteur non plus', async () => {
    const { result } = await mountOn()
    act(() => result.current.setTransportPlaying(false))
    engine.sync.mockClear()
    act(() => result.current.tick(5_000))
    expect(engine.sync).not.toHaveBeenCalled()
  })

  it('la reprise de la lecture rend aux moteurs leur suivi', async () => {
    const { result } = await mountOn()
    act(() => result.current.setTransportPlaying(false))
    act(() => result.current.setTransportPlaying(true))
    engine.sync.mockClear()
    act(() => result.current.tick(5_100))
    expect(engine.sync).toHaveBeenCalledWith(5_100)
  })
})

describe('useReplaySound — la pause éteint tout ce qui est en vol', () => {
  it('un son d événement en cours est arrêté par la rampe courte, pas à la fin de son fichier', async () => {
    const { result } = await mountOn()
    act(() => result.current.setTransportPlaying(true))
    act(() => result.current.tick(1_900))
    act(() => result.current.tick(2_050)) // le kill part
    expect(ctx.sources).toHaveLength(1)
    act(() => result.current.setTransportPlaying(false))
    expect(ctx.sources[0].stopped).toBeCloseTo(ctx.currentTime + VOLUME_RAMP_S)
  })

  it('à l arrêt, un pas d image joue encore le son bref qui atterrit avec l image', async () => {
    const { result } = await mountOn()
    act(() => result.current.setTransportPlaying(false))
    act(() => result.current.tick(1_900))
    act(() => result.current.tick(2_050))
    expect(ctx.sources).toHaveLength(1)
  })
})

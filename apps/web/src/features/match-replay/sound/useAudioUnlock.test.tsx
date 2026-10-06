/**
 * useAudioUnlock.test.tsx — LE SON DÉJÀ ACTIVÉ REVIT AU PREMIER GESTE SUR LA PAGE (item 12 du
 * plan backlog du 2026-09-26, décision D-10), jugé à travers `useReplaySound` : c'est là que
 * l'utilisateur l'entend, ou pas.
 *
 * LE DÉFAUT MESURÉ (A2.0, Chromium et Firefox, politique d'autoplay par défaut) : avec la lecture
 * automatique, la lecture démarre sans « Lecture » ni « Recommencer » ; aucun lecteur ne naît,
 * et un clic ailleurs sur la page n'y change rien. Le rejeu reste muet jusqu'à ce qu'on touche
 * au bouton du son. En passant d'un rejeu à un autre avec la lecture automatique, même silence,
 * alors que le document a déjà reçu un geste.
 *
 * On simule un RECHARGEMENT : la préférence est à « activé » dans le stockage local, et rien
 * n'est né.
 */
import { act, fireEvent, renderHook } from '@testing-library/react'

vi.mock('@/features/settings/queries', () => ({ useSettings: () => ({ data: undefined }) }))
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import type { ReplayKill } from '../model/killFeedLogic'
import { type FakeContext, flushAudio, installFakeAudio } from '../test/fakeAudio'
import { testReplayDoc } from '../test/testDoc'
import { useReplaySound, type ReplaySoundContext } from './useReplaySound'

const NO_CONTEXT: ReplaySoundContext = { allegiance: NO_ALLEGIANCE, endMatch: null, locale: undefined }

let ctx: FakeContext
let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  const fake = installFakeAudio()
  ctx = fake.ctx
  fetchMock = fake.fetchMock
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  // `userActivation` est posé sur l'instance par certains tests : on le retire.
  Reflect.deleteProperty(navigator, 'userActivation')
})

/** Un kill au BR à 2 000 ms : de quoi entendre (ou non) un son. */
function kill(): ReplayKill {
  return {
    replayMs: 2_000, medals: [],
    tMs: 2_000, xuid: 'K', teamID: 0,
    weaponKey: 'hinf_br75', weaponLabel: 'BR75', weaponImageUrl: '', weaponTinted: false,
    assistState: '', assistGamertag: '', assistTeamID: null,
    killerDamagePct: null, assistDamagePct: null,
    victimXuid: 'V', victimGamertag: 'Victime', victimTeamID: 1,
  }
}

function mount() {
  const doc = testReplayDoc({
    frameIntervalMs: 100,
    tracks: [
      { slot: 1, team: -1, xuid: 'K', points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 10, y: 0 }], startFrame: 0, endFrame: 100 },
      { slot: 2, team: -1, xuid: 'V', points: [{ t: 0, x: 5, y: 5 }, { t: 20, x: 5, y: 4 }], startFrame: 0, endFrame: 20 },
    ],
  })
  const kills = [kill()]
  return renderHook(() => useReplaySound(doc, kills, 1, NO_CONTEXT))
}

/** Le document a-t-il déjà reçu un geste ? (API `navigator.userActivation`, absente de jsdom.) */
function documentAlreadyActive(active: boolean) {
  Object.defineProperty(navigator, 'userActivation', { value: { hasBeenActive: active }, configurable: true })
}

/** Le kill passe : un son part si, et seulement si, un lecteur vit. */
function playKill(result: { current: ReturnType<typeof useReplaySound> }) {
  act(() => result.current.tick(1_900))
  act(() => result.current.tick(2_050))
}

describe('useAudioUnlock — préférence « activé », aucun lecteur (rechargement)', () => {
  beforeEach(() => localStorage.setItem('replay-sound-on', 'true'))

  it('le premier CLIC n’importe où sur la page ouvre le lecteur, une seule fois', async () => {
    const { result } = mount()
    fireEvent.click(document.body)
    await act(async () => { await flushAudio() })
    expect(ctx.resumed).toBe(1)
    expect(fetchMock).toHaveBeenCalledTimes(2) // le kill + l'extrait d'intro (item 7)
    playKill(result)
    expect(ctx.sources).toHaveLength(1)
    fireEvent.click(document.body)
    fireEvent.keyUp(document.body, { key: 'x' })
    expect(ctx.resumed).toBe(1)
  })

  it('une TOUCHE aussi (au relâchement) ouvre le lecteur', async () => {
    const { result } = mount()
    fireEvent.keyUp(document.body, { key: 'x' })
    await act(async () => { await flushAudio() })
    expect(ctx.resumed).toBe(1)
    playKill(result)
    expect(ctx.sources).toHaveLength(1)
  })

  it('l’écouteur est RETIRÉ après le premier geste', async () => {
    const removed = vi.spyOn(document, 'removeEventListener')
    mount()
    fireEvent.click(document.body)
    await act(async () => { await flushAudio() })
    const types = removed.mock.calls.map(([type]) => type)
    expect(types).toEqual(expect.arrayContaining(['click', 'keyup']))
  })

  it('document DÉJÀ activé à l’affichage (autre rejeu sans rechargement) : le lecteur s’ouvre sans geste', async () => {
    documentAlreadyActive(true)
    const { result } = mount()
    await act(async () => { await flushAudio() })
    expect(ctx.resumed).toBe(1)
    playKill(result)
    expect(ctx.sources).toHaveLength(1)
  })

  it('document jamais activé : RIEN avant le geste (politique d’autoplay)', () => {
    documentAlreadyActive(false)
    const { result } = mount()
    playKill(result)
    expect(ctx.gains).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  /**
   * NON-RÉGRESSION DU 2026-08-27 : le premier clic sur le BOUTON DU SON active, il ne coupe pas.
   * Un vrai clic émet `pointerdown` puis `click` ; le bouton (React, sous la racine) voit son
   * `click` AVANT l'écouteur du document. Un écouteur sur `pointerdown` ouvrirait le lecteur
   * AVANT la bascule, qui le trouverait vivant et COUPERAIT : deux clics pour entendre, le
   * défaut du 27/08 revenu par une autre porte.
   */
  it('le premier clic sur le BOUTON DU SON active et ne coupe pas', async () => {
    const { result } = mount()
    fireEvent.pointerDown(document.body)
    act(() => result.current.toggle()) // le `click` du bouton, sous la racine React
    fireEvent.click(document.body) // puis le même `click`, arrivé au document
    await act(async () => { await flushAudio() })
    expect(result.current.on).toBe(true)
    expect(localStorage.getItem('replay-sound-on')).toBe('true')
    expect(ctx.resumed).toBe(1)
    playKill(result)
    expect(ctx.sources).toHaveLength(1)
  })

  it('la touche M (raccourci du son, sur `keydown`) active et ne coupe pas', async () => {
    const { result } = mount()
    fireEvent.keyDown(document.body, { key: 'm' })
    act(() => result.current.toggle()) // le raccourci, écouté par la fenêtre sur `keydown`
    fireEvent.keyUp(document.body, { key: 'm' })
    await act(async () => { await flushAudio() })
    expect(result.current.on).toBe(true)
    expect(ctx.resumed).toBe(1)
  })

  it('démonté avant tout geste : un clic ensuite n’ouvre rien', () => {
    const { unmount } = mount()
    unmount()
    fireEvent.click(document.body)
    expect(ctx.gains).toHaveLength(0)
  })
})

describe('useAudioUnlock — préférence « coupé »', () => {
  it('ni geste ni document déjà activé n’ouvrent quoi que ce soit, rien ne se télécharge', async () => {
    documentAlreadyActive(true)
    const { result } = mount()
    fireEvent.click(document.body)
    fireEvent.keyUp(document.body, { key: 'x' })
    await act(async () => { await flushAudio() })
    playKill(result)
    expect(result.current.on).toBe(false)
    expect(ctx.gains).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })
})

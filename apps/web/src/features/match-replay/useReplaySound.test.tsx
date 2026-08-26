/**
 * useReplaySound.test.tsx — LE CÂBLAGE, là où un son se déclenche ou se tait pour de bon.
 *
 * Les règles pures ont leurs tests (replaySound.test.ts) et le moteur les siens
 * (replayAudio.test.ts). Reste ce que seul l'assemblage peut trahir : que rien ne sonne ni
 * ne se télécharge tant que l'utilisateur n'a pas cliqué, qu'activer le son en plein match
 * ne déverse pas d'un coup tout ce qui précède, qu'une avance rapide se taise sans perdre
 * le fil, et que la préférence survive à la page.
 */
import { act, renderHook } from '@testing-library/react'

// Les reglages d'instance (variation RANGED, distance) viennent de useSettings ; ici on
// les neutralise — leurs effets ont leurs propres tests (weaponSoundLogic, replayAudio).
vi.mock('@/features/settings/queries', () => ({ useSettings: () => ({ data: undefined }) }))
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'

import { SOUND_MAX_SPEED } from './replaySoundCursor'
import { type FakeContext, flushAudio, installFakeAudio } from './test/fakeAudio'
import { testReplayDoc } from './test/testDoc'
import { SOUND_VOLUME_DEFAULT, useReplaySound } from './useReplaySound'

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
})

function kill(over: Partial<KillEvent> = {}): KillEvent {
  return {
    tMs: 2_000, xuid: 'K', ally: true, teamID: 0,
    weaponKey: 'hinf_br75', weaponLabel: 'BR75', weaponImageUrl: '', weaponTinted: false,
    assistState: '', assistGamertag: '', assistTeamID: null,
    killerDamagePct: null, assistDamagePct: null,
    victimXuid: 'V', victimGamertag: 'Victime', victimTeamID: 1,
    ...over,
  }
}

/** Document 10 Hz avec un couple tueur/victime : la fin de vie tombe à 2 000 ms. */
function docWithCouple() {
  return testReplayDoc({
    frameIntervalMs: 100,
    tracks: [
      { slot: 1, team: -1, xuid: 'K', points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 10, y: 0 }], startFrame: 0, endFrame: 100 },
      { slot: 2, team: -1, xuid: 'V', points: [{ t: 0, x: 5, y: 5 }, { t: 20, x: 5, y: 4 }], startFrame: 0, endFrame: 20 },
    ],
  })
}

/** Le hook, monté sur un match d'un seul kill au BR à 2 000 ms. */
function mount(speed = 1) {
  const doc = docWithCouple()
  const kills = [kill()]
  return renderHook(({ s }: { s: number }) => useReplaySound(doc, kills, 0, s), {
    initialProps: { s: speed },
  })
}

describe('useReplaySound — coupé par défaut', () => {
  it('ne crée AUCUN contexte audio et ne télécharge rien tant qu on n a pas cliqué', () => {
    const { result } = mount()
    expect(result.current.on).toBe(false)
    expect(result.current.available).toBe(true)
    act(() => result.current.tick(2_500))
    expect(ctx.sources).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('pas un seul son dans la piste : aucune commande à offrir', () => {
    const doc = docWithCouple()
    const { result } = renderHook(() => useReplaySound(doc, [], 0, 1))
    expect(result.current.available).toBe(false)
  })
})

describe('useReplaySound — activation', () => {
  it('le clic ouvre le contexte et précharge les sons DE CE MATCH, pas le pack entier', async () => {
    const { result } = mount()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(result.current.on).toBe(true)
    expect(ctx.resumed).toBe(1)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(String(fetchMock.mock.calls[0][0])).toBe('/static/sounds/halo_infinite/hinf_br75.wav')
  })

  it('activer en plein match ne déverse pas ce qui précède : le premier battement recale', async () => {
    const { result } = mount()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    // Le kill (2 000 ms) est DÉJÀ passé quand le son s'active à 2 100 ms.
    act(() => result.current.tick(2_100))
    expect(ctx.sources).toHaveLength(0)
  })

  it('puis le son suivant part, une seule fois, à son passage', async () => {
    const { result } = mount()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    act(() => result.current.tick(1_900)) // recalage silencieux avant le kill
    act(() => result.current.tick(2_050)) // le kill passe ici
    expect(ctx.sources).toHaveLength(1)
    act(() => result.current.tick(2_150)) // rien de neuf : rien ne rejoue
    expect(ctx.sources).toHaveLength(1)
  })
})

describe('useReplaySound — silences voulus', () => {
  it('avance rapide : le son se tait, et revenir à 1x ne déclenche pas de salve', async () => {
    const { result, rerender } = mount(SOUND_MAX_SPEED * 2)
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(result.current.mutedBySpeed).toBe(true)
    act(() => result.current.tick(1_900))
    act(() => result.current.tick(2_050)) // le kill passe pendant l'avance rapide
    expect(ctx.sources).toHaveLength(0)
    rerender({ s: 1 })
    expect(result.current.mutedBySpeed).toBe(false)
    act(() => result.current.tick(2_150)) // le curseur a suivi : rien à rattraper
    expect(ctx.sources).toHaveLength(0)
  })

  it('la coupure est immédiate : le maître tombe à zéro, on n attend pas la fin des sons', async () => {
    const { result } = mount()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    act(() => result.current.toggle())
    const master = ctx.gains[0]
    expect(master.gain.calls.filter((c) => c[0] === 'ramp').pop()?.[1]).toBe(0)
    act(() => result.current.tick(1_900))
    act(() => result.current.tick(2_050))
    expect(ctx.sources).toHaveLength(0)
  })
})

describe('useReplaySound — catégories (tiroir de réglages, phase 2)', () => {
  it('les cinq catégories sont actives par défaut', () => {
    const { result } = mount()
    expect(result.current.categories).toEqual({
      weapon: true, grenade: true, melee: true, equipment: true, objective: true,
    })
  })

  it('couper ARMES retire le kill d arme de la piste jouée sans faire disparaître le panneau', async () => {
    const { result } = mount()
    act(() => result.current.toggleCategory('weapon'))
    expect(result.current.categories.weapon).toBe(false)
    expect(result.current.available).toBe(true) // le match A du son, même toute catégorie coupée
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    act(() => result.current.tick(1_900))
    act(() => result.current.tick(2_050)) // le kill (arme) passerait ici
    expect(ctx.sources).toHaveLength(0)
  })

  it('couper une catégorie retire SES sons, les autres restent inchangés', async () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      tracks: [
        { slot: 1, team: -1, xuid: 'K', points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 10, y: 0 }], startFrame: 0, endFrame: 100 },
        { slot: 2, team: -1, xuid: 'V', points: [{ t: 0, x: 5, y: 5 }, { t: 20, x: 5, y: 4 }], startFrame: 0, endFrame: 20 },
      ],
      grenades: [{ i: 0, rank: 0, s: '', slot: 1, t: 5, x: 0, y: 0 }], // throw_frag à 500 ms
    })
    const { result } = renderHook(() => useReplaySound(doc, [kill()], 0, 1))
    act(() => result.current.toggleCategory('weapon'))
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    act(() => result.current.tick(100)) // recalage silencieux, AVANT le lancer
    act(() => result.current.tick(600)) // le lancer (grenade, catégorie active) est passé
    expect(ctx.sources).toHaveLength(1)
    act(() => result.current.tick(2_100)) // le kill (arme, catégorie coupée) est passé
    expect(ctx.sources).toHaveLength(1) // toujours un seul son : l'arme reste muette
  })

  it('la préférence de catégorie survit au remontage (localStorage, comme le son)', () => {
    const first = mount()
    act(() => first.result.current.toggleCategory('equipment'))
    first.unmount()

    const { result } = mount()
    expect(result.current.categories.equipment).toBe(false)
    expect(result.current.categories.weapon).toBe(true)
  })
})

describe('useReplaySound — préférences', () => {
  it('l état du son et le volume survivent au remontage (localStorage)', async () => {
    const first = mount()
    act(() => first.result.current.toggle())
    act(() => first.result.current.setVolume(0.3))
    await act(async () => { await flushAudio() })
    first.unmount()
    expect(ctx.closed).toBe(1) // le contexte meurt avec le composant

    const { result } = mount()
    expect(result.current.on).toBe(true)
    expect(result.current.volume).toBe(0.3)
  })

  it('volume par défaut au premier passage, et bornes respectées', () => {
    const { result } = mount()
    expect(result.current.volume).toBe(SOUND_VOLUME_DEFAULT)
    act(() => result.current.setVolume(5))
    expect(result.current.volume).toBe(1)
    act(() => result.current.setVolume(-2))
    expect(result.current.volume).toBe(0)
  })
})

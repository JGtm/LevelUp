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

import type { EndMatchSoundSpec } from './endMatchSound'
import { SOUND_MAX_SPEED } from './replaySoundCursor'
import { type FakeContext, flushAudio, installFakeAudio } from './test/fakeAudio'
import { testReplayDoc } from './test/testDoc'
import { SOUND_VOLUME_DEFAULT, useReplaySound } from './useReplaySound'
import { WEAPON_BURST_SPECS } from './weaponBurstSpecs'

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

  /**
   * ET LA BASCULE REFUSE DE S'EXÉCUTER (correctif du 2026-08-28, revue R1).
   *
   * LE DÉFAUT QUE CE CAS FIXE : le bouton du son ne se rend pas sur un match muet, mais le
   * RACCOURCI CLAVIER « M », lui, n'a pas de rendu à respecter. Il basculait donc la
   * préférence, la PERSISTAIT dans le stockage local et ouvrait un AudioContext, sans qu'un
   * seul pixel ne change à l'écran — et le rejeu SUIVANT, celui-là sonore, démarrait dans
   * l'état inverse de celui qu'on croyait avoir laissé. La garde vit chez le propriétaire de
   * l'état, donc elle vaut pour tous les appelants, clavier compris.
   */
  it('MATCH MUET : la bascule ne persiste rien et n’ouvre aucun contexte audio', () => {
    const doc = docWithCouple()
    const { result } = renderHook(() => useReplaySound(doc, [], 0, 1))
    act(() => result.current.toggle())
    expect(result.current.on).toBe(false)
    expect(localStorage.getItem('replay-sound-on')).toBeNull()
    expect(ctx.sources).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
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

/**
 * LA RAFALE, VUE DU CÂBLAGE (lot C du 2026-08-27) — le moteur a ses propres tests
 * (replayAudio.test.ts) ; ce qui se joue ici est l'AIGUILLAGE : que le stem tiré décide, et
 * que les armes hors table gardent leur unique départ.
 */
describe('useReplaySound — rafale des armes automatiques', () => {
  /** Un match d'un seul TIR à 500 ms, de l'arme demandée. */
  function docWithShot(weaponKey: string) {
    return testReplayDoc({
      frameIntervalMs: 100,
      tracks: [
        { slot: 1, team: -1, xuid: 'K', points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 10, y: 0 }], startFrame: 0, endFrame: 100 },
      ],
      shots: [{ t: 5, slot: 1, x: 0, y: 0, w: '0xAA' }],
      weaponLabels: { '0xAA': { en: 'arme', fr: 'arme', key: weaponKey } },
    })
  }

  async function tickPastShot(doc: ReturnType<typeof docWithShot>) {
    const { result } = renderHook(() => useReplaySound(doc, [], 0, 1))
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    act(() => result.current.tick(100)) // recalage silencieux, AVANT le tir
    act(() => result.current.tick(600)) // le tir est passé
  }

  it('un tir de MA40 part en TROIS balles échelonnées (retour utilisateur du 2026-08-27)', async () => {
    await tickPastShot(docWithShot('hinf_ma40_ar'))
    expect(ctx.sources).toHaveLength(3)
    const t0 = ctx.currentTime
    const gap = WEAPON_BURST_SPECS.hinf_ma40_ar.ecartMs / 1000
    expect(ctx.sources.map((s) => s.started)).toEqual([t0, t0 + gap, t0 + 2 * gap])
  })

  it('une arme HORS table garde son unique départ (le chemin d avant, intact)', async () => {
    await tickPastShot(docWithShot('hinf_s7_sniper'))
    expect(ctx.sources).toHaveLength(1)
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

/**
 * LE RETOUR SUR UNE PAGE DONT LE SON ÉTAIT DÉJÀ ACTIVÉ (correctif du 2026-08-27).
 *
 * LE DÉFAUT QUE CES CAS FIXENT, signalé à l'usage : la préférence revient du stockage local, le
 * lecteur non (un AudioContext ne naît que dans un geste). Le panneau annonçait donc « son
 * activé » sur un rejeu muet, et le clic suivant — le seul geste qui pouvait tout réparer —
 * basculait la préférence à « coupé ». Il fallait DEUX clics pour entendre quoi que ce soit.
 *
 * `localStorage` est posé À LA MAIN plutôt que par un premier montage : ce qu'on veut
 * reproduire est un RECHARGEMENT DE PAGE, c'est-à-dire une préférence sans lecteur — un premier
 * montage laisserait derrière lui un contexte audio déjà ouvert et testerait autre chose.
 */
describe('useReplaySound — le son déjà activé revit au premier geste', () => {
  /** L'état exact d'un rechargement : la préférence dit « activé », rien n'est encore né. */
  function reloadWithSoundOn() {
    localStorage.setItem('replay-sound-on', 'true')
    return mount()
  }

  it('au chargement, la préférence est là mais RIEN ne sonne — c’est le désaccord à réparer', () => {
    const { result } = reloadWithSoundOn()
    expect(result.current.on).toBe(true)
    act(() => result.current.tick(1_900))
    act(() => result.current.tick(2_050)) // le kill passe ici
    expect(ctx.gains).toHaveLength(0) // aucun lecteur : pas même un gain maître
    expect(ctx.sources).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('LE PREMIER CLIC ACTIVE : le son part, et la préférence RESTE à « activé »', async () => {
    const { result } = reloadWithSoundOn()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(result.current.on).toBe(true)
    expect(localStorage.getItem('replay-sound-on')).toBe('true')
    expect(ctx.resumed).toBe(1)
    act(() => result.current.tick(1_900))
    act(() => result.current.tick(2_050))
    expect(ctx.sources).toHaveLength(1)
  })

  it('le clic SUIVANT coupe, comme d’habitude : la bascule n’est pas cassée', async () => {
    const { result } = reloadWithSoundOn()
    act(() => result.current.toggle()) // celui-ci active
    await act(async () => { await flushAudio() })
    act(() => result.current.toggle()) // celui-ci coupe
    expect(result.current.on).toBe(false)
    expect(localStorage.getItem('replay-sound-on')).toBe('false')
    act(() => result.current.tick(1_900))
    act(() => result.current.tick(2_050))
    expect(ctx.sources).toHaveLength(0)
  })

  it('un geste de transport suffit : le son revient sans passer par le bouton', async () => {
    const { result } = reloadWithSoundOn()
    act(() => result.current.wake())
    await act(async () => { await flushAudio() })
    expect(result.current.on).toBe(true)
    act(() => result.current.tick(1_900))
    act(() => result.current.tick(2_050))
    expect(ctx.sources).toHaveLength(1)
  })

  it('son COUPÉ : un geste de transport n’ouvre aucun contexte (la doctrine tient)', () => {
    const { result } = mount() // préférence par défaut : coupé
    act(() => result.current.wake())
    expect(ctx.gains).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('un geste de transport de plus ne recrée rien : le lecteur né reste le seul', async () => {
    const { result } = reloadWithSoundOn()
    act(() => result.current.wake())
    await act(async () => { await flushAudio() })
    const gains = ctx.gains.length
    act(() => result.current.wake())
    act(() => result.current.wake())
    expect(ctx.gains).toHaveLength(gains)
    expect(ctx.resumed).toBe(1)
  })

  it('la CONCLUSION sonore en bénéficie : un rejeu réveillé qui atteint la fin sonne', async () => {
    localStorage.setItem('replay-sound-on', 'true')
    const doc = docWithCouple()
    const { result } = renderHook(() =>
      useReplaySound(doc, [kill()], 0, 1, undefined, { outcome: 'win', ffa: false, locale: 'fr' }),
    )
    act(() => result.current.wake())
    await act(async () => { await flushAudio() })
    act(() => result.current.endMatch())
    expect(ctx.sources).toHaveLength(2) // la voix et la fanfare
  })
})

/**
 * LA FIN DE PARTIE (lot C, 2026-08-27). Les règles de SÉLECTION ont leurs tests
 * (endMatchSound.test.ts) ; ici on ne juge que le câblage — que la conclusion obéisse aux mêmes
 * silences que le reste (son coupé, avance rapide), que ses prises soient DÉJÀ chargées quand
 * elle part (un fichier demandé à l'arrivée sonnerait après le silence), et que la voix et la
 * fanfare partent bien ENSEMBLE, deux voix du lecteur et non un fichier pré-mixé.
 */
describe('useReplaySound — la fin de partie', () => {
  const VICTOIRE_FR: EndMatchSoundSpec = { outcome: 'win', ffa: false, locale: 'fr' }

  /** Le hook, monté sur le même match d'un kill, avec une fin de partie à annoncer. */
  function mountWithEnd(spec: EndMatchSoundSpec | null = VICTOIRE_FR) {
    const doc = docWithCouple()
    const kills = [kill()]
    return renderHook(() => useReplaySound(doc, kills, 0, 1, undefined, spec))
  }

  it('son coupé : la conclusion ne sonne pas, et n’ouvre aucun contexte au passage', () => {
    const { result } = mountWithEnd()
    act(() => result.current.endMatch())
    expect(ctx.sources).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('les prises sont préchargées AVEC la piste, tirage compris', async () => {
    const { result } = mountWithEnd()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    const charges = fetchMock.mock.calls.map((c) => String(c[0]))
    expect(charges).toContain('/static/sounds/halo_infinite/end_victory_voice_fr_01.wav')
    expect(charges).toContain('/static/sounds/halo_infinite/end_victory_voice_fr_02.wav')
    expect(charges).toContain('/static/sounds/halo_infinite/end_victory_music_01.wav')
    expect(charges).toContain('/static/sounds/halo_infinite/hinf_br75.wav')
  })

  it('à l’arrivée, la voix et la fanfare partent ensemble — deux voix, pas une', async () => {
    const { result } = mountWithEnd()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    act(() => result.current.endMatch())
    expect(ctx.sources).toHaveLength(2)
  })

  it('avance rapide : la conclusion se tait aussi, comme l’annonce le panneau', async () => {
    const doc = docWithCouple()
    const { result } = renderHook(() =>
      useReplaySound(doc, [kill()], 0, SOUND_MAX_SPEED * 2, undefined, VICTOIRE_FR),
    )
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(result.current.mutedBySpeed).toBe(true)
    act(() => result.current.endMatch())
    expect(ctx.sources).toHaveLength(0)
  })

  it('fin non lisible : rien à charger, rien à jouer — le reste du son est intact', async () => {
    const { result } = mountWithEnd(null)
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(fetchMock).toHaveBeenCalledTimes(1) // le seul son du match : le kill au BR
    act(() => result.current.endMatch())
    expect(ctx.sources).toHaveLength(0)
  })
})

/**
 * LA PISTE POUR LA VIDÉO (décision 6 du plan de capture). Ce que ces cas tiennent : un rejeu
 * muet ne fabrique AUCUN nœud audio — le son est coupé par défaut, et filmer un match ne doit
 * pas ouvrir un contexte que personne n'a demandé.
 */
describe('useReplaySound — la piste d’enregistrement', () => {
  it('son coupé : pas de piste, et surtout aucun contexte ouvert au passage', () => {
    const { result } = mount()
    expect(result.current.recordingTrack()).toBeNull()
    expect(ctx.streamDests).toHaveLength(0)
  })

  it('son activé : la piste existe et vient du lecteur en cours', async () => {
    const { result } = mount()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    expect(result.current.recordingTrack()).toBe(ctx.streamDests[0].track)
  })

  it('couper le son après coup rend de nouveau `null`', async () => {
    // Le clip EN COURS garde sa piste (elle est câblée au démarrage) : ce que ce cas dit,
    // c'est qu'un enregistrement lancé APRÈS la coupure repart muet.
    const { result } = mount()
    act(() => result.current.toggle())
    await act(async () => { await flushAudio() })
    act(() => result.current.toggle())
    expect(result.current.recordingTrack()).toBeNull()
  })
})
